package dlog

import (
	"context"
	"dlog/config"
	"dlog/utils"
	"io"
	"io/ioutil"
	"os"
	"sync"
)

type Dlog struct {
	wg          *sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	file        *os.File
	fetcher     *Fetcher
	initialized bool
}

// GetFile returns input file, original or cache file
func (d *Dlog) GetFile() *os.File { return d.file }

func (d *Dlog) Init() {
	d.fetcher = NewFetcher(d.ctx, d.file)
	d.initialized = true
}

func (d *Dlog) Display() {
	if !d.initialized {
		d.Init()
	}

	_, _ = d.file.Seek(0, io.SeekStart)
	d.fetcher.seek(0)

	// Viewer
}

func (d *Dlog) Shutdown() {
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

func NewFromFile(path string) (*Dlog, error) {
	if err := utils.ValidateRegularFile(path); err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return New(f), nil
}

func makeCacheFile() (f *os.File, err error) {
	if config.Config.CachePath == "" {
		f, err = ioutil.TempFile(os.TempDir(), "dlog_")
	} else {
		f = utils.OpenRewrite(config.Config.CachePath)
	}

	return
}
