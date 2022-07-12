package dlog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/dimcz/dlog/logging"
	"github.com/dimcz/dlog/memfile"

	"github.com/docker/docker/pkg/stdcopy"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const MinimalRecordsChunk = "100"
const TimeShift = -24

type Container struct {
	ID   string
	Name string
}

type Docker struct {
	out        *memfile.File
	in         *memfile.File
	containers []Container
	current    int
	reader     io.ReadCloser
	cli        *client.Client

	wg            *sync.WaitGroup
	parentContext context.Context
	ctx           context.Context
	cancel        func()
}

func (d *Docker) followFrom(t time.Time) {
	defer d.wg.Done()

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
	d.ctx, d.cancel = context.WithCancel(d.parentContext)

	logging.Debug(fmt.Sprintf("request %s first records", MinimalRecordsChunk))
	start, end, err := d.retrieveAndParseLogs(types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: true,
		Tail:       MinimalRecordsChunk,
	})
	if err != nil {
		logging.Debug("failed to execute retrieveLogs:", err)
		return
	}

	logging.Debug("execute following process")
	d.wg.Add(1)
	go d.followFrom(end.Add(1))

	logging.Debug("execute append process")
	d.wg.Add(1)
	go d.appendSince(start.Add(-1))
}

func (d *Docker) appendSince(t time.Time) {
	defer d.wg.Done()
	defer logging.Timeit("append logs")()

	end := t.Add(-1)
	var start time.Time

	for {
		select {
		case <-d.ctx.Done():
			return
		default:
			start = end.Add(time.Duration(TimeShift) * time.Hour)
			logging.Debug(fmt.Sprintf("request block between %s and %s", start, end))
			_, err := d.retrieveLogs(types.ContainerLogsOptions{
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
		parentContext: ctx,
		out:           out,
		in:            in,
		cli:           cli,
		containers:    containers,
		wg:            new(sync.WaitGroup),
	}, nil
}

func retrieveContainers(cli *client.Client) (containers []Container, err error) {
	defer logging.Timeit("retrieveContainers")()

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
	return fmt.Sprintf("(%d/%d) %s (ID:%s)",
		d.current+1,
		len(d.containers),
		strings.Replace(d.containers[d.current].Name, "/", "", 1),
		d.containers[d.current].ID[:12])
}

func (d *Docker) getNextContainer() {
	d.cancel()
	d.wg.Wait()

	c := d.current + 1
	if c >= len(d.containers) {
		c = 0
	}
	d.current = c
}

func (d *Docker) getPrevContainer() {
	d.cancel()
	d.wg.Wait()

	c := d.current - 1
	if c < 0 {
		c = len(d.containers) - 1
	}
	d.current = c
}

func (d *Docker) retrieveLogs(options types.ContainerLogsOptions) (*memfile.File, error) {
	fd, err := d.cli.ContainerLogs(d.ctx, d.containers[d.current].ID, options)
	if err != nil {
		return nil, err
	}

	defer func(fd io.ReadCloser) {
		_ = fd.Close()
	}(fd)

	mf := memfile.New([]byte{})

	w, err := stdcopy.StdCopy(mf, mf, fd)
	if err != nil {
		return nil, err
	}

	if len(mf.Bytes()) == 0 {
		return nil, fmt.Errorf("retrieve empty logs")
	}

	logging.Debug(fmt.Sprintf("retrieveLogs: got buffer array with length %d, write %d", len(mf.Bytes()), w))

	if _, err := d.out.Insert(mf.Bytes()); err != nil {
		return nil, err
	}

	d.in.SetLen(d.out.GetLen())

	logging.Debug(fmt.Sprintf("retrieveLogs: after insert, array length is %d", len(mf.Bytes())))

	return mf, nil
}

func (d *Docker) retrieveAndParseLogs(opts types.ContainerLogsOptions) (time.Time, time.Time, error) {
	mf, err := d.retrieveLogs(opts)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	str := strings.Split(string(mf.Bytes()[0:bytes.IndexByte(mf.Bytes(), '\n')]), " ")[0]

	start, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	index := bytes.LastIndex(mf.Bytes(), []byte{'\n'})
	index = bytes.LastIndex(mf.Bytes()[0:index-1], []byte{'\n'})

	str = strings.Split(string(mf.Bytes()[index+1:]), " ")[0]
	end, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return start, end, nil
}
