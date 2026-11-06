// Package storage mendefinisikan port penyimpanan file.
// Sengaja memakai io.Reader (bukan *multipart.FileHeader) supaya usecase
// tidak terikat ke HTTP dan mudah di-unit test.
package storage

import (
	"context"
	"io"
)

// Upload adalah file masuk yang siap disimpan.
type Upload struct {
	FileName string
	Size     int64
	Content  io.Reader
}

// Stored adalah hasil penyimpanan file.
type Stored struct {
	FileName string // nama asli
	Path     string // path relatif hasil simpan
}

// Content adalah file yang dibaca kembali dari penyimpanan.
// Pemanggil wajib menutup Reader.
type Content struct {
	Reader io.ReadCloser
	Size   int64
}

type FileStorage interface {
	Save(ctx context.Context, upload Upload) (*Stored, error)
	Open(ctx context.Context, path string) (*Content, error)
	Delete(ctx context.Context, path string) error
}
