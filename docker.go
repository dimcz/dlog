package dlog

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"dlog/logging"
	"dlog/memfile"

	"github.com/docker/docker/pkg/stdcopy"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const TimeShift = -24 * 7

type Container struct {
	ID   string
	Name string
}

type Docker struct {
	ctx        context.Context
	out        *memfile.File
	in         *memfile.File
	containers []Container
	current    int
	reader     io.ReadCloser
	cli        *client.Client
}

func (d *Docker) retrieveLogs(opts types.ContainerLogsOptions) error {
	fd, err := d.cli.ContainerLogs(d.ctx, d.containers[d.current].ID, opts)
	if err != nil {
		log.Fatal(err)
	}

	defer func(fd io.ReadCloser) {
		_ = fd.Close()
	}(fd)

	mf := memfile.New([]byte{})

	w, err := stdcopy.StdCopy(mf, mf, fd)
	if err != nil {
		return err
	}

	if len(mf.Bytes()) == 0 {
		return fmt.Errorf("retrieve empty logs")
	}

	logging.Debug(fmt.Sprintf("retrieveLogs: got buffer array with length %d, write %d", len(mf.Bytes()), w))

	if _, err := d.out.Insert(mf.Bytes()); err != nil {
		return err
	}

	d.in.SetLen(d.out.GetLen())

	logging.Debug(fmt.Sprintf("retrieveLogs: after insert, array length is %d", len(mf.Bytes())))

	return nil
}

func (d *Docker) follow(t time.Time) {
	if d.reader != nil {
		_ = d.reader.Close()
	}

	var err error

	logging.Debug(fmt.Sprintf("request block from %s", t.Add(1).Format(time.RFC3339)))

	d.reader, err = d.cli.ContainerLogs(d.ctx, d.containers[d.current].ID, types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Follow:     true,
		Timestamps: true,
		Since:      t.Add(1).Format(time.RFC3339),
	})
	if err != nil {
		return
	}

	if _, err := stdcopy.StdCopy(d.out, d.out, d.reader); err != nil {
		return
	}
}

func (d *Docker) Close() error {
	if d.reader != nil {
		return d.reader.Close()
	}

	return nil
}

func (d *Docker) logs() {
	now := time.Now()
	t := now.Add(time.Duration(TimeShift) * time.Hour)

	logging.Debug("execute retrieveLogs")
	logging.Debug(fmt.Sprintf("request block between %s and %s", t, now))
	err := d.retrieveLogs(types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: true,
		Since:      t.Format(time.RFC3339),
		Until:      now.Format(time.RFC3339),
	})
	if err != nil {
		logging.Debug("failed to execute retrieveLogs:", err)
		return
	}

	logging.Debug("execute following process")
	go d.follow(now.Add(1))

	logging.Debug("execute append process")
	d.append(t)
}

func (d *Docker) append(t time.Time) {
	end := t.Add(-1)
	var start time.Time

	for {
		start = end.Add(time.Duration(TimeShift) * time.Hour)
		logging.Debug(fmt.Sprintf("request block between %s and %s", start, end))
		err := d.retrieveLogs(types.ContainerLogsOptions{
			ShowStderr: true,
			ShowStdout: true,
			Timestamps: true,
			Until:      end.Format(time.RFC3339),
			Since:      start.Format(time.RFC3339),
		})
		if err != nil {
			logging.Debug("failed to execute retrieveLogs:", err)
			return
		}
		end = start.Add(-1)
	}
}

func DockerClient(ctx context.Context, out, in *memfile.File) (*Docker, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	if _, err := cli.Ping(context.Background()); err != nil {
		return nil, err
	}

	containers, err := retrieveContainers(cli)
	if err != nil {
		return nil, err
	}

	return &Docker{
		ctx:        ctx,
		out:        out,
		in:         in,
		cli:        cli,
		containers: containers,
	}, nil
}

func retrieveContainers(cli *client.Client) (containers []Container, err error) {
	list, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	for _, c := range list {
		containers = append(containers, Container{c.ID, strings.Join(c.Names, ", ")})
	}

	return containers, nil
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
}

func (d *Docker) getPrevContainer() {
	c := d.current - 1
	if c < 0 {
		c = len(d.containers) - 1
	}
	d.current = c
}
