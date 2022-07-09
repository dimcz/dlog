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
}

func (d *Dlog) GetFile() *os.File {
	return d.file
}

func (d *Dlog) Display() {
	d.fetcher = NewFetcher(d.ctx, d.file)
	_, _ = d.file.Seek(0, io.SeekStart)
	d.fetcher.seek(0)

	v := &viewer{
		fetcher:   d.fetcher,
		ctx:       d.ctx,
		wrap:      true,
		keepChars: 0,
	}
	v.termGui()
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

	d.docker, err = DockerSetup()
	if err != nil {
		return nil, err
	}

	d.wg.Add(1)
	go d.docker.LoadLogs(d.wg, cacheFile)

	return d, nil
}
