package dlog

import (
	"context"
	"dlog/logging"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type Container struct {
	ID   string
	Name string
}

type Docker struct {
	containers []Container
	current    int
	reader     io.ReadCloser
}

func docker() *client.Client {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	return cli
}

func CheckDaemon() bool {
	if _, err := docker().Ping(context.Background()); err != nil {
		logging.Debug(err)

		return false
	}

	return true
}

func DockerSetup() (*Docker, error) {
	ctx := context.Background()

	containers, err := docker().ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	d := Docker{}

	for _, c := range containers {
		d.containers = append(d.containers, Container{
			ID:   c.ID,
			Name: strings.Join(c.Names, ", "),
		})
	}

	return &d, nil
}

func (d *Docker) Close() error {
	if d.reader != nil {
		return d.reader.Close()
	}

	return nil
}

func (d *Docker) LoadLogs(wg *sync.WaitGroup, out io.Writer) {
	defer wg.Done()

	if d.reader != nil {
		_ = d.reader.Close()
	}

	var err error

	d.reader, err = docker().ContainerLogs(context.Background(), d.containers[d.current].ID, types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: true,
		Follow:     true,
	})

	if err != nil {
		return
	}

	for {
		if _, err := stdcopy.StdCopy(out, out, d.reader); err != nil {
			return
		}
	}
}
