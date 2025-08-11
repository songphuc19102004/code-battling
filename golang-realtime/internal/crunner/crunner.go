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

type RunResult struct {
	Result          Result
	TotalTestCases  int
	TestCasesPassed int
	FailedTestCase  any
	Log             string
	ExitCode        int
}

func NewDockerClient() *client.Client {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	return cli
}
