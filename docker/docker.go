package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dimcz/dlog/config"
	"github.com/dimcz/dlog/logging"
	"github.com/dimcz/dlog/memfile"

	"github.com/docker/docker/pkg/stdcopy"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type Container struct {
	ID   string
	Name string
}

type Docker struct {
	file       *memfile.File
	containers []Container
	current    int
	cli        *client.Client

	wg            *sync.WaitGroup
	parentContext context.Context
	ctx           context.Context
	cancel        func()
}

func (d *Docker) followFrom(t int64) {
	defer d.wg.Done()

	logging.Debug("request block from", t)

	fd, err := d.cli.ContainerLogs(d.ctx, d.containers[d.current].ID, types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Follow:     true,
		Timestamps: true,
		Since:      strconv.FormatInt(t+1, 10),
	})
	if err != nil {
		return
	}

	defer func(fd io.ReadCloser) {
		logging.LogOnErr(fd.Close())
	}(fd)

	if _, err := stdcopy.StdCopy(d.file, d.file, fd); err != nil {
		return
	}
}

func (d *Docker) Follow() int64 {
	d.ctx, d.cancel = context.WithCancel(d.parentContext)

	h := strconv.Itoa(config.GetValue().Tail)

	d.file.Clear()

	logging.Debug(fmt.Sprintf("request %s first records", h))
	start, end, err := d.retrieveAndParseLogs(types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: true,
		Tail:       h,
	})
	if err != nil {
		logging.Debug("failed to execute retrieveLogs:", err)
		return -1
	}

	logging.Debug("execute following process")
	d.wg.Add(1)
	go d.followFrom(end)

	return start
}

func (d *Docker) Append(start int64, callBack func()) {
	if !config.GetValue().NoLoad {
		logging.Debug("execute append process")
		d.wg.Add(1)
		go d.appendSince(start, callBack)
	}
}

func (d *Docker) appendSince(t int64, callBack func()) {
	defer d.wg.Done()
	defer logging.Timeit("append logs")()

	end := t - 1
	var start int64

	for {
		select {
		case <-d.ctx.Done():
			return
		default:
			start = end - config.GetValue().TimeShift
			_, err := d.retrieveLogs(types.ContainerLogsOptions{
				ShowStderr: true,
				ShowStdout: true,
				Timestamps: true,
				Until:      strconv.FormatInt(end, 10),
				Since:      strconv.FormatInt(start, 10),
			})
			if err != nil {
				logging.Debug("failed to execute retrieveLogs:", err)
				return
			}
			end = start - 1

			callBack()
		}
	}
}

func Client(ctx context.Context, file *memfile.File) (*Docker, error) {
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
		file:          file,
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

func (d *Docker) Name() string {
	return fmt.Sprintf("(%d/%d) %s (ID:%s)",
		d.current+1,
		len(d.containers),
		strings.Replace(d.containers[d.current].Name, "/", "", 1),
		d.containers[d.current].ID[:12])
}

func (d *Docker) NextContainer() {
	d.cancel()
	d.wg.Wait()

	c := d.current + 1
	if c >= len(d.containers) {
		c = 0
	}
	d.current = c
}

func (d *Docker) PrevContainer() {
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
		logging.LogOnErr(fd.Close())
	}(fd)

	mf := memfile.New([]byte{})

	_, err = stdcopy.StdCopy(mf, mf, fd)
	if err != nil {
		return nil, err
	}

	if len(mf.Bytes()) == 0 {
		return nil, fmt.Errorf("retrieve empty logs")
	}

	if _, err := d.file.Insert(mf.Bytes()); err != nil {
		return nil, err
	}

	return mf, nil
}

func (d *Docker) retrieveAndParseLogs(opts types.ContainerLogsOptions) (int64, int64, error) {
	mf, err := d.retrieveLogs(opts)
	if err != nil {
		return -1, -1, err
	}

	str := strings.Split(string(mf.Bytes()[0:bytes.IndexByte(mf.Bytes(), '\n')]), " ")[0]

	start, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return -1, -1, err
	}

	index := bytes.LastIndex(mf.Bytes(), []byte{'\n'})
	index = bytes.LastIndex(mf.Bytes()[0:index-1], []byte{'\n'})

	str = strings.Split(string(mf.Bytes()[index+1:]), " ")[0]
	end, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return -1, -1, err
	}

	return start.Unix(), end.Unix(), nil
}
