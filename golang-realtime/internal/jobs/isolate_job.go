package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang-realtime/internal/store"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	stdinFileName              = "stdin.txt"
	stdoutFileName             = "stdout.txt"
	stderrFileName             = "stderr.txt"
	metadataFileName           = "metadata.txt"
	additionalFilesArchiveName = "additional_files.zip"
)

type IsolateJob struct {
	submission  *store.Submission
	language    *store.Language
	config      *Config
	queries     *store.Queries
	cgroupsFlag string
	boxID       int
	workdir     string
	boxdir      string
	tmpdir      string

	sourceFile                 string
	stdinFile                  string
	stdoutFile                 string
	stderrFile                 string
	metadataFile               string
	additionalFilesArchiveFile string
}

func NewIsolateJob(submission *store.Submission, language *store.Language, config *Config, queries *store.Queries) *IsolateJob {
	return &IsolateJob{
		submission: submission,
		language:   language,
		config:     config,
		queries:    queries,
	}
}

func (j *IsolateJob) Perform() (err error) {
	log.Printf("Starting processing for submission token %s (ID: %d)", j.submission.Token.String, j.submission.ID)

	hostname, _ := os.Hostname()
	j.submission.StatusID.Int32 = StatusProcessing
	j.submission.StartedAt.Time = time.Now()
	j.submission.ExecutionHost.String = hostname

	if err := j.saveSubmission(); err != nil {
		return err
	}

	defer j.callCallback()
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic: %v", r)
			log.Printf("ERROR: Panic during submission %d processing: %v", j.submission.ID, r)
			j.submission.Message.String = fmt.Sprintf("%v", r)
			j.submission.StatusID.Int32 = StatusInternalError
			j.submission.FinishedAt.Time = time.Now().UTC()
			j.saveSubmission() // Attempt to save error state
		}
		if err != nil {
			log.Printf("ERROR: Failed to process submission %d: %v", j.submission.ID, err)
			j.submission.Message.String = fmt.Sprintf("%v", err)
			j.submission.StatusID.Int32 = StatusInternalError
			j.submission.FinishedAt.Time = time.Now().UTC()
			j.saveSubmission()
		}
		j.cleanup(false) // false = do not panic on cleanup failure
	}()

	return nil
}

// --- Private Helper Methods ---

// initializeWorkdir sets up the isolate sandbox directory and necessary files.
func (j *IsolateJob) initializeWorkdir() error {
	j.boxID = int(j.submission.ID % 2147483647)

	// Determine cgroups flag based on submission settings
	if !j.submission.EnablePerProcessAndThreadTimeLimit.Bool || !j.submission.EnablePerProcessAndThreadMemoryLimit.Bool {
		j.cgroupsFlag = "--cg"
	} else {
		j.cgroupsFlag = ""
	}

	cmd := exec.Command("isolate", j.cgroupsFlag, "-b", strconv.Itoa(j.boxID), "--init")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("isolate init failed: %s - %w", string(output), err)
	}
	j.workdir = strings.TrimSpace(string(output))
	j.boxdir = filepath.Join(j.workdir, "box")
	j.tmpdir = filepath.Join(j.workdir, "tmp")

	j.sourceFile = filepath.Join(j.boxdir, j.language.SourceFile.String)
	j.stdinFile = filepath.Join(j.workdir, stdinFileName)
	j.stdoutFile = filepath.Join(j.workdir, stdoutFileName)
	j.stderrFile = filepath.Join(j.workdir, stderrFileName)
	j.metadataFile = filepath.Join(j.workdir, metadataFileName)
	j.additionalFilesArchiveFile = filepath.Join(j.boxdir, additionalFilesArchiveName)

	filesToInit := []string{j.stdinFile, j.stdoutFile, j.stderrFile, j.metadataFile}
	for _, f := range filesToInit {
		if err := j.initializeFile(f); err != nil {
			return fmt.Errorf("failed to initialize file %s: %w", f, err)
		}
	}

	if !languageIsProject(*j.language) && j.submission.SourceCode.Valid {
		if err := os.WriteFile(j.sourceFile, []byte(j.submission.SourceCode.String), 0644); err != nil {
			return fmt.Errorf("failed to write source code: %w", err)
		}
	}

	if j.submission.Stdin.Valid {
		if err := os.WriteFile(j.stdinFile, []byte(j.submission.Stdin.String), 0644); err != nil {
			return fmt.Errorf("failed to write stdin: %w", err)
		}
	}

	return j.extractArchive()
}

// initializeFile creates a file and sets its ownership.
func (j *IsolateJob) initializeFile(path string) error {
	if _, err := os.Create(path); err != nil {
		return err
	}
	// In Go, we assume the user running the worker can write to the created files.
	// The `sudo chown` from the Ruby script might be necessary depending on Docker setup.
	// For simplicity, we omit it here as file creation is usually sufficient.
	return nil
}

// extractArchive unpacks additional_files inside the sandbox if they exist.
func (j *IsolateJob) extractArchive() error {
	if len(j.submission.AdditionalFiles) == 0 {
		return nil
	}

	if err := os.WriteFile(j.additionalFilesArchiveFile, j.submission.AdditionalFiles, 0644); err != nil {
		return fmt.Errorf("failed to write additional files archive: %w", err)
	}

	args := j.buildIsolateCmdArgs(isolateArgs{
		timeLimit:      2,
		extraTime:      1,
		wallTime:       4,
		stackLimit:     j.config.MaxStackLimit,
		processes:      j.config.MaxMaxProcessesAndOrThreads,
		memLimit:       j.config.MaxMemoryLimit,
		fileSize:       j.config.MaxExtractSize,
		run:            true,
		stderrToStdout: true,
	})
	args = append(args, "--", "/usr/bin/unzip", "-n", "-qq", additionalFilesArchiveName)

	cmd := exec.Command("isolate", args...)
	log.Printf("Executing archive extraction: %s", cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unzip command failed: %s - %w", string(output), err)
	}

	return os.Remove(j.additionalFilesArchiveFile)
}

// compile runs the language's compile command inside the sandbox.
func (j *IsolateJob) compile() (bool, error) {
	if languageIsProject(*j.language) {
		// For project-based submissions, check if a compile script exists.
		compileScriptPath := filepath.Join(j.boxdir, "compile")
		if _, err := os.Stat(compileScriptPath); os.IsNotExist(err) {
			compileScriptPath = filepath.Join(j.boxdir, "compile.sh")
			if _, err := os.Stat(compileScriptPath); os.IsNotExist(err) {
				return true, nil // No compile script, so compilation is successful by default.
			}
		}
	} else if !j.language.CompileCmd.Valid {
		return true, nil // Not a compiled language.
	}

	compileScriptPath := filepath.Join(j.boxdir, "compile.sh")
	if !languageIsProject(*j.language) {
		// Sanitize compiler options (simple version)
		sanitizedOpts := strings.ReplaceAll(j.submission.CompilerOptions.String, "`", "")
		sanitizedOpts = strings.ReplaceAll(sanitizedOpts, "$", "")
		// ... add more sanitization as needed

		compileCmdStr := fmt.Sprintf(j.language.CompileCmd.String, sanitizedOpts)
		if err := os.WriteFile(compileScriptPath, []byte(compileCmdStr), 0755); err != nil {
			return false, fmt.Errorf("failed to write compile script: %w", err)
		}
	}

	compileOutputFile := filepath.Join(j.workdir, "compile_output.txt")
	if err := j.initializeFile(compileOutputFile); err != nil {
		return false, fmt.Errorf("failed to init compile output file: %w", err)
	}

	args := j.buildIsolateCmdArgs(isolateArgs{
		metaFile:       j.metadataFile,
		timeLimit:      j.config.MaxCPUTimeLimit,
		extraTime:      0,
		wallTime:       j.config.MaxWallTimeLimit,
		stackLimit:     j.config.MaxStackLimit,
		processes:      j.config.MaxMaxProcessesAndOrThreads,
		memLimit:       j.config.MaxMemoryLimit,
		fileSize:       j.config.MaxMaxFileSize,
		stdinFile:      "/dev/null",
		run:            true,
		stderrToStdout: true,
		fullEnv:        true,
	})
	args = append(args, "--", "/bin/bash", filepath.Base(compileScriptPath))

	cmd := exec.Command("isolate", args...)

	outFile, err := os.Create(compileOutputFile)
	if err != nil {
		return false, fmt.Errorf("failed to create compile output file: %w", err)
	}
	defer outFile.Close()
	cmd.Stdout = outFile

	log.Printf("Executing compile: %s", cmd.String())
	err = cmd.Run()
	// An exit error is expected if compilation fails; we process it below.
	if err != nil {
		log.Printf("Compile command finished with error: %v", err)
	}

	compileOutput, readErr := os.ReadFile(compileOutputFile)
	if readErr != nil {
		return false, fmt.Errorf("failed to read compile output: %w", readErr)
	}
	if len(compileOutput) > 0 {
		j.submission.CompileOutput.String = string(compileOutput)
	}

	metadata, metaErr := j.getMetadata()
	if metaErr != nil {
		return false, fmt.Errorf("failed to get compile metadata: %w", metaErr)
	}

	j.resetMetadataFile()

	os.Remove(compileOutputFile)
	if !languageIsProject(*j.language) {
		os.Remove(compileScriptPath)
	}

	if cmd.ProcessState.Success() {
		return true, nil
	}

	// Compilation failed
	if status, ok := metadata["status"]; ok && status == "TO" {
		j.submission.CompileOutput.String = fmt.Sprint("Compilation time limit exceeded")
	}

	j.submission.FinishedAt.Time = time.Now().UTC()
	j.submission.StatusID.Int32 = StatusCompilation
	j.submission.Time = pgtype.Numeric{}
	j.submission.WallTime = pgtype.Numeric{}
	j.submission.Memory = pgtype.Int4{}
	j.submission.Stdout = pgtype.Text{}
	j.submission.Stderr = pgtype.Text{}
	j.submission.ExitCode = pgtype.Int4{}
	j.submission.ExitSignal = pgtype.Int4{}
	j.submission.Message = pgtype.Text{}
	if err := j.saveSubmission(); err != nil {
		return false, fmt.Errorf("failed to save compilation error: %w", err)
	}

	return false, nil
}

// run executes the submission's code inside the sandbox.
func (j *IsolateJob) run() error {
	runScriptPath := filepath.Join(j.boxdir, "run.sh")

	if !languageIsProject(*j.language) {
		// Sanitize command line arguments
		sanitizedArgs := strings.ReplaceAll(j.submission.CommandLineArguments.String, "`", "")
		sanitizedArgs = strings.ReplaceAll(sanitizedArgs, "$", "")
		// ... add more sanitization

		runCmdStr := fmt.Sprintf("%s %s", j.language.RunCmd, sanitizedArgs)
		if err := os.WriteFile(runScriptPath, []byte(runCmdStr), 0755); err != nil {
			return fmt.Errorf("failed to write run script: %w", err)
		}
	} else {
		// For project-based submissions, check for "run" or "run.sh"
		if _, err := os.Stat(runScriptPath); os.IsNotExist(err) {
			runScriptPath = filepath.Join(j.boxdir, "run")
			if _, err := os.Stat(runScriptPath); os.IsNotExist(err) {
				return fmt.Errorf("run script not found for project submission")
			}
		}
	}

	args := j.buildIsolateCmdArgs(isolateArgs{
		metaFile:       j.metadataFile,
		timeLimit:      float64(j.submission.CpuTimeLimit.Exp),
		extraTime:      float64(j.submission.CpuTimeLimit.Exp),
		wallTime:       float64(j.submission.WallTimeLimit.Exp),
		stackLimit:     int(j.submission.StackLimit.Int32),
		processes:      int(j.submission.MaxProcessesAndOrThreads.Int32),
		memLimit:       int(j.submission.MemoryLimit.Int32),
		fileSize:       int(j.submission.MaxFileSize.Int32),
		run:            true,
		shareNet:       j.submission.EnableNetwork.Bool,
		fullEnv:        true,
		stderrToStdout: j.submission.RedirectStderrToStdout.Bool,
	})
	args = append(args, "--", "/bin/bash", filepath.Base(runScriptPath))

	cmd := exec.Command("isolate", args...)

	stdin, err := os.Open(j.stdinFile)
	if err != nil {
		return fmt.Errorf("failed to open stdin file: %w", err)
	}
	defer stdin.Close()
	cmd.Stdin = stdin

	stdout, err := os.Create(j.stdoutFile)
	if err != nil {
		return fmt.Errorf("failed to create stdout file: %w", err)
	}
	defer stdout.Close()
	cmd.Stdout = stdout

	stderr, err := os.Create(j.stderrFile)
	if err != nil {
		return fmt.Errorf("failed to create stderr file: %w", err)
	}
	defer stderr.Close()
	cmd.Stderr = stderr

	log.Printf("Executing run: %s", cmd.String())
	err = cmd.Run()
	if err != nil {
		log.Printf("Run command finished with error: %v", err)
	}

	if !languageIsProject(*j.language) {
		os.Remove(runScriptPath)
	}

	return nil
}

// verify reads the outputs and metadata to determine the final status of the submission.
func (j *IsolateJob) verify() error {
	j.submission.FinishedAt.Time = time.Now().UTC()

	metadata, err := j.getMetadata()
	if err != nil {
		return fmt.Errorf("failed to get run metadata: %w", err)
	}

	stdout, err := os.ReadFile(j.stdoutFile)
	if err != nil {
		return fmt.Errorf("failed to read stdout: %w", err)
	}
	if len(stdout) > 0 {
		j.submission.Stdout.String = string(stdout)
	} else {
		j.submission.Stdout = pgtype.Text{}
	}

	stderr, err := os.ReadFile(j.stderrFile)
	if err != nil {
		return fmt.Errorf("failed to read stderr: %w", err)
	}
	if len(stderr) > 0 {
		j.submission.Stderr.String = string(stderr)
	} else {
		j.submission.Stderr = pgtype.Text{}
	}

	if val, ok := metadata["time"]; ok {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			j.submission.Time = pgtype.Numeric{
				Int: big.NewInt(i),
			}
		}
	}
	if val, ok := metadata["time-wall"]; ok {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			j.submission.WallTime = pgtype.Numeric{
				Int: big.NewInt(i),
			}
		}
	}

	memKey := "max-rss"
	if j.cgroupsFlag != "" {
		memKey = "cg-mem"
	}
	if val, ok := metadata[memKey]; ok {
		if i, err := strconv.ParseInt(val, 10, 32); err == nil {
			j.submission.Memory = pgtype.Int4{
				Int32: int32(i),
			}
		}
	}

	if val, ok := metadata["exitcode"]; ok {
		if i, err := strconv.ParseInt(val, 10, 32); err == nil {
			j.submission.ExitCode = pgtype.Int4{
				Int32: int32(i),
			}
		}
	}
	if val, ok := metadata["exitsig"]; ok {
		if i, err := strconv.ParseInt(val, 10, 32); err == nil {
			j.submission.ExitSignal = pgtype.Int4{
				Int32: int32(i),
			}
		}
	}
	if val, ok := metadata["message"]; ok {
		j.submission.Message = pgtype.Text{
			String: val,
		}
	}

	status, _ := metadata["status"]
	j.submission.StatusID.Int32 = j.determineStatus(status, int64(j.submission.ExitSignal.Int32))

	if j.submission.StatusID.Int32 == StatusInternalError && (strings.Contains(j.submission.Message.String, "Exec format error") || strings.Contains(j.submission.Message.String, "No such file or directory") || strings.Contains(j.submission.Message.String, "Permission denied")) {
		j.submission.StatusID.Int32 = StatusExecFormat
	}

	return nil
}

func (j *IsolateJob) saveSubmission() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s, err := j.queries.GetSubmission(ctx, j.submission.ID)
	if err != nil {
		return err
	}

	s, err = j.queries.UpdateSubmission(ctx, s.ToUpdateParam())
	return nil
}

type CallbackPayload struct {
	Token         string         `json:"token"`
	Time          float64        `json:"time,string"`
	Memory        int64          `json:"memory"`
	Stdout        string         `json:"stdout"`
	Stderr        string         `json:"stderr"`
	CompileOutput string         `json:"compile_output"`
	Message       string         `json:"message"`
	Status        CallbackStatus `json:"status"`
}

type CallbackStatus struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
}

func (j *IsolateJob) callCallback() {
	if !j.submission.CallbackUrl.Valid || j.submission.CallbackUrl.String == "" {
		return
	}

	// NOTE: The Ruby version forces base64 encoding and a specific set of fields for the callback.
	// We replicate that here.
	payload := CallbackPayload{
		Token:         j.submission.Token.String,
		Time:          float64(j.submission.Time.Int.Int64()),
		Memory:        int64(j.submission.Memory.Int32),
		Stdout:        j.submission.Stdout.String, // Assume base64 is handled elsewhere or not required for this logic.
		Stderr:        j.submission.Stderr.String,
		CompileOutput: j.submission.CompileOutput.String,
		Message:       j.submission.Message.String,
		Status: CallbackStatus{
			ID:          int64(j.submission.StatusID.Int32),
			Description: getStatusDescription(int64(j.submission.StatusID.Int32)),
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ERROR: Failed to marshal callback payload for submission %d: %v", j.submission.ID, err)
		return
	}

	client := &http.Client{
		Timeout: j.config.CallbacksTimeout,
	}

	for i := 0; i < j.config.CallbacksMaxTries; i++ {
		req, err := http.NewRequest(http.MethodPut, j.submission.CallbackUrl.String, bytes.NewBuffer(payloadBytes))
		if err != nil {
			log.Printf("ERROR: Failed to create callback request for submission %d: %v", j.submission.ID, err)
			time.Sleep(1 * time.Second) // Wait before retrying
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("ERROR: Callback attempt %d for submission %d failed: %v", i+1, j.submission.ID, err)
			time.Sleep(1 * time.Second) // Wait before retrying
			continue
		}

		resp.Body.Close()
		log.Printf("Callback for submission %d sent successfully with status %d", j.submission.ID, resp.StatusCode)
		return // Success
	}
	log.Printf("ERROR: All callback attempts for submission %d failed.", j.submission.ID)
}

// cleanup removes the sandbox directory.
func (j *IsolateJob) cleanup(panicOnError bool) {
	if j.workdir == "" {
		return
	}

	// First fix permissions so we can remove files
	chownCmd := exec.Command("sudo", "chown", "-R", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()), j.boxdir)
	chownCmd.Run() // Best effort

	// Best effort removal of files outside the box
	os.RemoveAll(filepath.Join(j.boxdir, "*"))
	os.RemoveAll(filepath.Join(j.tmpdir, "*"))
	os.Remove(j.stdinFile)
	os.Remove(j.stdoutFile)
	os.Remove(j.stderrFile)
	os.Remove(j.metadataFile)

	cmd := exec.Command("isolate", j.cgroupsFlag, "-b", strconv.Itoa(j.boxID), "--cleanup")
	log.Printf("Executing cleanup: %s", cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := fmt.Sprintf("cleanup of sandbox %d failed: %s - %v", j.boxID, string(output), err)
		if panicOnError {
			panic(msg)
		} else {
			log.Printf("ERROR: %s", msg)
		}
	}
	j.workdir = "" // Prevent double cleanup
}

// getMetadata parses the key-value metadata file from isolate.
func (j *IsolateJob) getMetadata() (map[string]string, error) {
	content, err := os.ReadFile(j.metadataFile)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]string)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			metadata[parts[0]] = parts[1]
		}
	}
	return metadata, nil
}

func (j *IsolateJob) resetMetadataFile() error {
	if err := os.Remove(j.metadataFile); err != nil {
		return err
	}
	return j.initializeFile(j.metadataFile)
}

// This could be wron with the number tyype
func (j *IsolateJob) determineStatus(status string, exitSignal int64) int32 {
	switch status {
	case "TO":
		return StatusTimeLimit
	case "SG":
		switch exitSignal {
		case int64(syscall.SIGSEGV):
			return StatusRTSigsegv
		case int64(syscall.SIGXFSZ):
			return StatusRTSigxfsz
		case int64(syscall.SIGFPE):
			return StatusRTSigfpe
		case int64(syscall.SIGABRT):
			return StatusRTSigabrt
		default:
			return StatusRTOther
		}
	case "RE":
		return StatusRTNzec
	case "XX":
		return StatusInternalError
	}

	if !j.submission.ExpectedOutput.Valid || strip(j.submission.ExpectedOutput.String) == strip(j.submission.Stdout.String) {
		return StatusAccepted
	}
	return StatusWrongAnswer
}

func strip(text string) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), " \t")
}

// getStatusDescription is a placeholder for getting status name from ID.
func getStatusDescription(id int64) string {
	// In a real app, this might come from a map or another DB lookup.
	switch id {
	case StatusInQueue:
		return "In Queue"
	case StatusProcessing:
		return "Processing"
	case StatusAccepted:
		return "Accepted"
	case StatusWrongAnswer:
		return "Wrong Answer"
	case StatusTimeLimit:
		return "Time Limit Exceeded"
	case StatusCompilation:
		return "Compilation Error"
	case StatusRTSigsegv:
		return "Runtime Error (SIGSEGV)"
	case StatusRTSigxfsz:
		return "Runtime Error (SIGXFSZ)"
	case StatusRTSigfpe:
		return "Runtime Error (SIGFPE)"
	case StatusRTSigabrt:
		return "Runtime Error (SIGABRT)"
	case StatusRTNzec:
		return "Runtime Error (NZEC)"
	case StatusRTOther:
		return "Runtime Error (Other)"
	case StatusInternalError:
		return "Internal Error"
	case StatusExecFormat:
		return "Exec Format Error"
	default:
		return "Unknown"
	}
}

type isolateArgs struct {
	metaFile       string
	timeLimit      float64
	extraTime      float64
	wallTime       float64
	stackLimit     int
	processes      int
	memLimit       int
	fileSize       int
	stdinFile      string
	run            bool
	shareNet       bool
	fullEnv        bool
	stderrToStdout bool
}

func (j *IsolateJob) buildIsolateCmdArgs(opts isolateArgs) []string {
	args := []string{j.cgroupsFlag, "-s", "-b", strconv.Itoa(j.boxID)}

	if opts.metaFile != "" {
		args = append(args, "-M", opts.metaFile)
	}
	if opts.timeLimit > 0 {
		args = append(args, "-t", fmt.Sprintf("%f", opts.timeLimit))
	}
	if opts.extraTime > 0 {
		args = append(args, "-x", fmt.Sprintf("%f", opts.extraTime))
	}
	if opts.wallTime > 0 {
		args = append(args, "-w", fmt.Sprintf("%f", opts.wallTime))
	}
	if opts.stackLimit > 0 {
		args = append(args, "-k", strconv.Itoa(opts.stackLimit))
	}
	if opts.processes > 0 {
		args = append(args, "-p"+strconv.Itoa(opts.processes))
	}
	if opts.memLimit > 0 {
		if j.submission.EnablePerProcessAndThreadMemoryLimit.Bool {
			args = append(args, "-m", strconv.Itoa(opts.memLimit))
		} else {
			args = append(args, "--cg-mem="+strconv.Itoa(opts.memLimit))
		}
	}
	if j.submission.EnablePerProcessAndThreadTimeLimit.Bool {
		if j.cgroupsFlag != "" {
			args = append(args, "--no-cg-timing")
		}
	} else {
		args = append(args, "--cg-timing")
	}
	if opts.fileSize > 0 {
		args = append(args, "-f", strconv.Itoa(opts.fileSize))
	}
	if opts.stdinFile != "" {
		args = append(args, "-i", opts.stdinFile)
	}
	if opts.stderrToStdout {
		args = append(args, "--stderr-to-stdout")
	}
	if opts.shareNet {
		args = append(args, "--share-net")
	}
	if opts.fullEnv {
		args = append(args,
			"-E", "HOME=/tmp",
			"-E", "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			"-E", "LANG", "-E", "LANGUAGE", "-E", "LC_ALL",
			"-E", "JUDGE0_HOMEPAGE", "-E", "JUDGE0_SOURCE_CODE", "-E", "JUDGE0_MAINTAINER", "-E", "JUDGE0_VERSION",
			"-d", "/etc:noexec")
	}
	if opts.run {
		args = append(args, "--run")
	}
	return args
}

func languageIsProject(l store.Language) bool {
	return l.Name.String == "Multi-file program"
}
