package storage

import (
	"context"
	"io"
)

type Upload struct {
	FileName string
	Size     int64
	Content  io.Reader
}

type Stored struct {
	FileName string // nama asli
	Path     string // path relatif hasil simpan
}

type Content struct {
	Reader io.ReadCloser
	Size   int64
}

type FileStorage interface {
	Save(ctx context.Context, upload Upload) (*Stored, error)
	Open(ctx context.Context, path string) (*Content, error)
	Delete(ctx context.Context, path string) error
}
