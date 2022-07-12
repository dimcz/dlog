package dlog

import (
	"context"
	"io"
	"sync"

	"github.com/dimcz/dlog/logging"
	"github.com/dimcz/dlog/memfile"
	"github.com/nsf/termbox-go"
)

type Dlog struct {
	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	file    *memfile.File
	fetcher *Fetcher
	docker  *Docker
	v       *viewer
}

func (d *Dlog) GetFile() *memfile.File {
	return d.file
}

func (d *Dlog) Display() {
	d.fetcher = NewFetcher(d.ctx, d.file)
	d.resetFetcher()

	d.v = NewViewer(
		WithCtx(d.ctx),
		WithFetcher(d.fetcher),
		WithWrap(true),
		WithKeyArrowRight(d.rightDirection),
		WithKeyArrowLeft(d.leftDirection))

	d.v.termGui(d.docker.getName())
}

func (d *Dlog) resetFetcher() {
	_, err := d.file.Seek(0, io.SeekStart)
	logging.LogOnErr(err)

	d.fetcher.seek(0)
}

func (d *Dlog) rightDirection() {
	d.v.initScreen()
	d.docker.getNextContainer()
	d.reload()
}

func (d *Dlog) leftDirection() {
	d.v.initScreen()
	d.docker.getPrevContainer()
	d.reload()
}

func (d *Dlog) reload() {
	logging.LogOnErr(d.docker.out.Truncate(0))

	d.v.setTerminalName(d.docker.getName())
	d.docker.logs()
	d.resetFetcher()
	d.v.navigateEnd()
}

func (d *Dlog) Shutdown() {
	d.cancel()
	d.wg.Wait()
}

func New(f *memfile.File) *Dlog {
	ctx, cancel := context.WithCancel(context.Background())

	return &Dlog{
		wg:     new(sync.WaitGroup),
		ctx:    ctx,
		cancel: cancel,
		file:   f,
	}
}

func NewWithDocker() (*Dlog, error) {
	if err := termbox.Init(); err != nil {
		return nil, err
	}

	_, initHeight := termbox.Size()
	termbox.Close()

	fWR := memfile.New([]byte{})

	fRO := memfile.NewWithBuffer(fWR.Buffer())
	d := New(fRO)

	docker, err := DockerClient(d.ctx, initHeight, fWR, fRO)
	if err != nil {
		return nil, err
	}

	d.docker = docker

	docker.logs()

	return d, nil
}
