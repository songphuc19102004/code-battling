// crunner means code runner, dummy
package crunner

import (
	"golang-realtime/internal/store"
	"log/slog"
)

type Result string

const (
	Success Result = "success"
	Failure Result = "failure"
)

type CRunner interface {
	Execute(i RunInput, logger *slog.Logger) (RunOutput, error)
}

type RunnerManagerOptions struct {
	MaxWorkers   int
	MemoryLimit  int
	MaxJobCount  int
	CpuNanoLimit int64
}

type RunInput struct {
	Code     string
	Language store.Language
}

type RunOutput struct {
	Result          Result
	TotalTestCases  int
	TestCasesPassed int
	FailedTestCase  any
	Log             string
	ExitCode        int
}
