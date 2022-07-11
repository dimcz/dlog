package dlog

import (
	"context"
	"io"
	"sync"

	"dlog/logging"
	"dlog/memfile"
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
	d.v.setTerminalName(d.docker.getName())
	d.docker.fetchLogs(d.wg)
	d.resetFetcher()
}

func (d *Dlog) Shutdown() {
	logging.LogOnErr(d.docker.Close())

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
	fWR := memfile.New([]byte{})
	dd, err := DockerSetup(fWR)
	if err != nil {
		return nil, err
	}

	fRO := memfile.NewWithBuffer(fWR.Buffer())
	d := New(fRO)
	d.docker = dd

	dd.fetchLogs(d.wg)

	return d, nil
}
