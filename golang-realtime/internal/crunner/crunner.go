// crunner means code runner, dummy
// work with Isolate
package crunner

import (
	"golang-realtime/internal/store"
	"log/slog"

	"github.com/docker/docker/client"
)

type Result string

const (
	Success Result = "success"
	Failure Result = "failure"
)

type CRunner interface {
	Run(i RunInput, logger *slog.Logger) (RunOutput, error)
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

func NewDockerClient() *client.Client {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	return cli
}
