// crunner means code runner, dummy
package crunner

import (
	"log/slog"

	"github.com/docker/docker/client"
)

type Result string

const (
	Success Result = "success"
	Failure Result = "failure"
)

type CRunner interface {
	Run(logger *slog.Logger) (RunResult, error)
}

type RunRequest struct {
	Language  string
	Code      string
	TestCases []string
}

type RunResult struct {
	Result   Result
	Log      string
	ExitCode int
}

func NewDockerClient() *client.Client {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	return cli
}
