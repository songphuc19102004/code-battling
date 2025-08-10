package crunner

import (
	"log/slog"

	"github.com/docker/docker/client"
)

type DockerRunner struct {
	cli *client.Client
}

func NewDockerRunner(cli *client.Client) *DockerRunner {
	return &DockerRunner{
		cli: cli,
	}
}

func (d *DockerRunner) Run(logger *slog.Logger) (RunResult, error) {
	logger.Info("runContainer() hit")

	// ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return RunResult{}, err
	}
	defer cli.Close()

	logger.Info("Spawning a job container")
	// runSimpleJobContainer(ctx, cli)

	result := RunResult{
		Result:   Failure,
		ExitCode: 0,
	}
	return result, nil
}

// func runSimpleJobContainer(ctx context.Context, cli *client.Client) {
// 	imageName := "alpine:latest"
// 	log.Printf("Pulling image: %s\n", imageName)

// 	reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
// 	if err != nil {
// 		panic(err)
// 	}

// 	io.Copy(os.Stdout, reader)
// 	reader.Close()
// 	log.Println("Creating job container...")

// 	resp, err := cli.ContainerCreate(ctx, &container.Config{
// 		Image: imageName,
// 		Cmd:   []string{"echo", "Hello from the job container!"},
// 		Tty:   false,
// 	}, &container.HostConfig{
// 		// IMPORTANT: Clean up the container after it exits
// 		AutoRemove: true,
// 	}, nil, nil, "") // Use an empty string for a random container name
// 	if err != nil {
// 		log.Printf("Failed to create container: %v", err)
// 		return
// 	}

// 	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
// 		log.Printf("Failed to start container: %v", err)
// 		return
// 	}

// 	log.Printf("Image pulled successfully")

// 	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
// 	select {
// 	case err := <-errCh:
// 		if err != nil {
// 			panic(err)
// 		}
// 	case <-statusCh:
// 	}

// 	out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
// 	if err != nil {
// 		panic(err)
// 	}

// 	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
// }
