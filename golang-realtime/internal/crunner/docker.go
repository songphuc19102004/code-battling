package crunner

import (
	"log/slog"
	"os"
	"path/filepath"
)

type DockerRunner struct {
	logger *slog.Logger
}

func NewDockerRunner(logger *slog.Logger) *DockerRunner {
	return &DockerRunner{
		logger: logger,
	}
}

// Run will create an isolate job and wait for the job to run
func (d *DockerRunner) Run(i RunInput, logger *slog.Logger) (RunOutput, error) {
	logger.Info("runContainer() hit")
	var o RunOutput

	dir, err := os.MkdirTemp("", "example")
	if err != nil {
		d.logger.Error("error making dir", "err", err)
		return o, err
	}

	defer os.RemoveAll(dir)

	file := filepath.Join(dir, "tmpfile")
	if err := os.WriteFile(file, []byte("content"), 0666); err != nil {
		d.logger.Error("error writing file",
			"err", err,
			"file", file,
		)

		return o, err
	}

	return o, nil
}
