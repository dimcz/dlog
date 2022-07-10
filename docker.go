package dlog

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"dlog/logging"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const InitialLogTails = 1000

type Container struct {
	ID   string
	Name string
}

type Docker struct {
	out        *os.File
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

func DockerSetup(out *os.File) (*Docker, error) {
	ctx := context.Background()

	containers, err := docker().ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	d := Docker{out: out}

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

func (d *Docker) fetchLogs(wg *sync.WaitGroup) {
	opts := types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: true,
	}

	wg.Add(1)
	go d.loadLogs(wg, opts)
}

func (d *Docker) loadLogs(wg *sync.WaitGroup, opts types.ContainerLogsOptions) {
	defer wg.Done()

	if d.reader != nil {
		logging.LogOnErr(d.reader.Close())
	}

	var err error

	opts.Follow = true
	// opts.Tail = strconv.Itoa(InitialLogTails)
	d.reader, err = docker().ContainerLogs(context.Background(), d.containers[d.current].ID, opts)

	if err != nil {
		return
	}

	for {
		if _, err := stdcopy.StdCopy(d.out, d.out, d.reader); err != nil {
			return
		}
	}
}

func (d *Docker) getName() string {
	return fmt.Sprintf("%s (ID:%s)",
		strings.Replace(d.containers[d.current].Name, "/", "", 1),
		d.containers[d.current].ID[:12])
}

func (d *Docker) getNextContainer() {
	c := d.current + 1
	if c >= len(d.containers) {
		c = 0
	}
	d.current = c

	if d.reader != nil {
		logging.LogOnErr(d.reader.Close())
	}

	logging.LogOnErr(d.out.Truncate(0))
}

func (d *Docker) getPrevContainer() {
	c := d.current - 1
	if c < 0 {
		c = len(d.containers) - 1
	}
	d.current = c

	if d.reader != nil {
		logging.LogOnErr(d.reader.Close())
	}

	logging.LogOnErr(d.out.Truncate(0))
}
