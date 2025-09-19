package executor

import (
	"bytes"
	"context"
	"fmt"
	"golang-realtime/internal/store"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	QueryTimeOutSecond   = 30 * time.Second
	CodeRunTimeOutSecond = 10 * time.Second
)

type Job struct {
	Language store.Language
	Code     string
	Input    *string
	Result   chan Result
}

type Result struct {
	Output        string
	Sucess        bool
	Error         error
	ExecutionTime string
}

type WorkerPool struct {
	cm           *DockerContainerManager
	queries      *store.Queries
	logger       *slog.Logger
	jobs         chan Job
	wg           sync.WaitGroup
	shutdownChan chan any
}

type WorkerPoolOptions struct {
	MaxWorkers       int
	MemoryLimitBytes int64
	MaxJobCount      int
	CpuNanoLimit     int64
}

func NewWorkerPool(logger *slog.Logger, queries *store.Queries, opts *WorkerPoolOptions) (*WorkerPool, error) {
	cm, err := NewDockerContainerManager(opts.MaxWorkers, opts.MemoryLimitBytes, opts.CpuNanoLimit)
	if err != nil {
		return nil, err
	}

	err = cm.InitializePool()
	if err != nil {
		return nil, err
	}

	w := &WorkerPool{
		cm:           cm,
		queries:      queries,
		logger:       logger,
		jobs:         make(chan Job, opts.MaxJobCount),
		shutdownChan: make(chan any),
	}

	for i := range opts.MaxWorkers {
		w.wg.Add(1)
		go w.worker(i + 1)
	}

	w.logger.Info("Initialized worker pool with max workers",
		"max_worker", w.cm.maxWorkers)

	return w, err
}

func (w *WorkerPool) worker(id int) {
	defer w.wg.Done()
	w.logger.Info("Worker started", "id", id)

	for {
		select {
		case j, ok := <-w.jobs:
			if !ok {
				w.logger.Info("Worker shutting down due to channel closed",
					"worker_id", id)
				return
			}
			w.executeJob(id, j)

		case <-w.shutdownChan:
			w.logger.Info("Worker received shutdown signal", "worker_id", id)
			return
		}
	}
}

// ExecuteJob submits the job for execution
// input as a pointer so we could either set it or make it null
func (w *WorkerPool) ExecuteJob(lang store.Language, code string, input *string) Result {
	w.logger.Info("Submitting job...",
		"language", lang)

	result := make(chan Result, 1)
	select {
	case w.jobs <- Job{Language: lang, Code: code, Input: input, Result: result}:
		return <-result
	default:
		w.logger.Warn("Job queue is full, rejecting job...",
			"language", lang,
			"maxJobCount", w.cm.maxWorkers)
		return Result{}
	}
}

// executeJob handle the execution of a *single* job
func (w *WorkerPool) executeJob(workerID int, job Job) error {
	w.logger.Info("Job has been picked",
		"worker_id", workerID,
		"job", job)
	containerID, err := w.cm.GetAvailableContainer()
	if err != nil {
		w.logger.Error("Failed to get available Container",
			"err", err)
		return err
	}

	err = w.cm.SetContainerState(containerID, StateBusy)
	if err != nil {
		return err
	}

	start := time.Now()
	output, success, err := w.executeCode(job.Language, containerID, job.Code, job.Input)
	duration := time.Since(start)

	if err != nil {
		w.logger.Error("Worker job failed",
			"worker_id", workerID,
			"container_id", containerID,
			"duration", duration.Milliseconds(),
			"lang", job.Language,
			"err", err)
	} else {
		err = w.cm.SetContainerState(containerID, StateIdle)
		w.logger.Info("Worker job completed",
			"worker_id", workerID,
			"container_id", containerID,
			"duration", duration.Milliseconds(),
			"lang", job.Language)
	}

	// send result to result channel
	job.Result <- Result{
		Output:        output,
		Sucess:        success,
		Error:         err,
		ExecutionTime: fmt.Sprintf("%dms", duration.Milliseconds()),
	}

	return nil
}

// executeCode run the code in a specific Container
func (w *WorkerPool) executeCode(lang store.Language, containerID, code string, input *string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeOutSecond)
	defer cancel()

	// TODO: add input
	var stdout, stderr bytes.Buffer
	runCmd := generateRunCmd(lang.RunCmd.String, code)

	// -i for interactive
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", containerID, "sh", "-c", runCmd)

	if input != nil {
		w.logger.Info("Input is not nil",
			"input", *input)
		cmd.Stdin = strings.NewReader(*input)
	}

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)
	if err != nil {
		w.logger.Error("Failed to execute code",
			"container_id", containerID,
			"duration", duration,
			"err", err,
			"stdout", stdout.String(),
			"stderr", stderr.String())
		return stderr.String(), false, err
	}

	w.logger.Info("Code Execution Completed",
		"container_id", containerID,
		"duration", duration)

	return stdout.String(), true, nil
}

// generateCodeRunCmd will generate a run command for the code
func generateRunCmd(runCmd, finalCode string) string {
	formattedCode := strings.ReplaceAll(finalCode, "'", "'\\''")
	return fmt.Sprintf(runCmd, formattedCode)
}
