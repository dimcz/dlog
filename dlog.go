package dlog

import (
	"context"
	"io"
	"sync"

	"github.com/dimcz/dlog/docker"
	"github.com/dimcz/dlog/memfile"
	"github.com/nsf/termbox-go"
)

type Dlog struct {
	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	file    *memfile.File
	fetcher *Fetcher
	docker  *docker.Docker
	v       *viewer
}

func (d *Dlog) GetFile() *memfile.File {
	return d.file
}

func (d *Dlog) Display() {
	start := d.docker.Follow(height())

	d.fetcher = NewFetcher(d.ctx, d.file)
	_, _ = d.file.Seek(0, io.SeekStart)

	d.v = NewViewer(
		WithCtx(d.ctx),
		WithFetcher(d.fetcher),
		WithWrap(true),
		WithKeyArrowRight(d.rightDirection),
		WithKeyArrowLeft(d.leftDirection))

	d.v.termGui(d.docker.Name(), func() {
		d.docker.Append(start, d.v.refill)
	})
}

func (d *Dlog) rightDirection() {
	d.v.initScreen()
	d.docker.NextContainer()
	d.reload()
}

func (d *Dlog) leftDirection() {
	d.v.initScreen()
	d.docker.PrevContainer()
	d.reload()
}

func (d *Dlog) reload() {
	start := d.docker.Follow(d.v.height)

	d.v.setTerminalName(d.docker.Name())
	d.v.navigateEnd()

	d.docker.Append(start, d.v.refill)
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
	memFile := memfile.New([]byte{})

	d := New(memFile)

	var err error

	d.docker, err = docker.Client(d.ctx, memFile)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func height() int {
	defer termbox.Close()

	if err := termbox.Init(); err != nil {
		panic(err)
	}

	_, h := termbox.Size()
	return h
}
