package dlog

import (
	"context"
	"os"
	"sync"
)

type Offset int64

type Fetcher struct {
	lock sync.RWMutex
}

func NewFetcher(ctx context.Context, file *os.File) *Fetcher {
	return &Fetcher{}
}

func (f *Fetcher) seek(offset Offset) {

}
