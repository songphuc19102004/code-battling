package executor

import (
	"bytes"
	"context"
	"fmt"
	service "golang-realtime/internal/services"
	"golang-realtime/internal/store"
	"log/slog"
	"os/exec"
	"sync"
	"time"
)

const (
	QueryTimeOutSecond   = 5 * time.Second
	CodeRunTimeOutSecond = 10 * time.Second
)

type Job struct {
	Language string
	Code     string
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
	MaxWorkers   int
	MemoryLimit  int
	MaxJobCount  int
	CpuNanoLimit int64
}

func NewWorkerPool(logger *slog.Logger, queries *store.Queries, opts *WorkerPoolOptions) (*WorkerPool, error) {
	cm, err := NewDockerContainerManager()
	if err != nil {
		return nil, err
	}
	w := &WorkerPool{
		cm:      cm,
		queries: queries,
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
func (w *WorkerPool) ExecuteJob(lang, code string) Result {
	w.logger.Info("Submitting job...",
		"language", lang)

	result := make(chan Result, 1)
	select {
	case w.jobs <- Job{Language: lang, Code: code, Result: result}:
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
	output, success, err := w.executeCode(containerID, job.Code, job.Language)
	duration := time.Since(start)

	if err != nil {
		w.logger.Error("Worker job failed",
			"worker_id", workerID,
			"container_id", containerID,
			"duration", duration.Milliseconds(),
			"lang", job.Language,
			"err", err)
	} else {
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
func (w *WorkerPool) executeCode(containerID, code, lang string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeOutSecond)
	defer cancel()

	normalized := service.NormalizeLanguage(lang)
	langCfg, err := w.queries.GetLanguageByName(ctx, normalized)
	if err != nil {
		w.logger.Error("Failed to get language config",
			"lang", lang,
			"err", err)
		return "", false, err
	}

	var output bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", append([]string{"exec", containerID}, langCfg.RunCmd.String)...)
	cmd.Stdout = &output
	cmd.Stderr = &output

	start := time.Now()
	err = cmd.Run()
	duration := time.Since(start)
	if err != nil {
		w.logger.Error("Failed to execute code",
			"container_id", containerID,
			"duration", duration,
			"err", err)
		return output.String(), false, err
	}

	w.logger.Info("Code Execution Completed",
		"container_id", containerID,
		"duration", duration)

	return output.String(), true, nil
}
