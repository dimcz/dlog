package dlog

import (
	"context"
	"io"
	"os"
	"sync"

	"dlog/config"
	"dlog/utils"
)

type Dlog struct {
	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	file    *os.File
	fetcher *Fetcher
	docker  *Docker
	v       *viewer
}

func (d *Dlog) GetFile() *os.File {
	return d.file
}

func (d *Dlog) Display() {
	d.InitFetcher()

	d.v = NewViewer(
		WithCtx(d.ctx),
		WithFetcher(d.fetcher),
		WithWrap(true),
		WithKeyArrowRight(d.rightDirection),
		WithKeyArrowLeft(d.leftDirection))

	d.v.termGui(d.docker.getName())
}

func (d *Dlog) InitFetcher() {
	d.fetcher = NewFetcher(d.ctx, d.file)
	_, _ = d.file.Seek(0, io.SeekStart)
	d.fetcher.seek(0)
}

func (d *Dlog) rightDirection() {
	d.v.initScreen()
	d.docker.getNextContainer()
	d.reloadFetcher()
}

func (d *Dlog) leftDirection() {
	d.v.initScreen()
	d.docker.getPrevContainer()
	d.reloadFetcher()
}

func (d *Dlog) reloadFetcher() {
	d.wg.Wait()
	d.v.setTerminalName(d.docker.getName())
	d.docker.fetchLogs(d.wg)
	d.InitFetcher()
}

func (d *Dlog) Shutdown() {
	_ = d.docker.Close()
	d.cancel()
	d.wg.Wait()
	_ = d.file.Close()
	_ = os.Remove(d.file.Name())
}

func New(f *os.File) *Dlog {
	ctx, cancel := context.WithCancel(context.Background())

	return &Dlog{
		wg:     new(sync.WaitGroup),
		ctx:    ctx,
		cancel: cancel,
		file:   f,
	}
}

func NewWithDocker() (*Dlog, error) {
	cacheFile, err := utils.MakeCacheFile(config.Config.CachePath)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(cacheFile.Name())
	if err != nil {
		return nil, err
	}

	d := New(f)

	d.docker, err = DockerSetup(cacheFile)
	if err != nil {
		return nil, err
	}

	d.docker.fetchLogs(d.wg)

	return d, nil
}
