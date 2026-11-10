package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	domainstorage "daily-task-go/internal/domain/storage"
	"daily-task-go/pkg/apperror"
)

type localStorage struct {
	baseDir    string
	maxBytes   int64
	allowedExt []string
}

func NewLocalStorage(baseDir string, maxBytes int64, allowedExt []string) (domainstorage.FileStorage, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("gagal menyiapkan folder upload: %w", err)
	}
	return &localStorage{baseDir: baseDir, maxBytes: maxBytes, allowedExt: allowedExt}, nil
}

func (s *localStorage) Save(ctx context.Context, upload domainstorage.Upload) (*domainstorage.Stored, error) {
	if err := ctx.Err(); err != nil {
		return nil, apperror.Internal("permintaan dibatalkan", err)
	}

	originalName := filepath.Base(strings.TrimSpace(upload.FileName))
	if originalName == "" || originalName == "." || originalName == string(filepath.Separator) {
		return nil, apperror.Validation("file tidak valid", map[string]string{"file": "nama file tidak valid"})
	}

	if s.maxBytes > 0 && upload.Size > s.maxBytes {
		return nil, apperror.Validation("file terlalu besar", map[string]string{
			"file": fmt.Sprintf("maksimal %d MB", s.maxBytes/(1024*1024)),
		})
	}

	ext := strings.ToLower(filepath.Ext(originalName))
	if len(s.allowedExt) > 0 && !slices.Contains(s.allowedExt, ext) {
		return nil, apperror.Validation("tipe file tidak diizinkan", map[string]string{
			"file": "ekstensi yang diizinkan: " + strings.Join(s.allowedExt, ", "),
		})
	}

	storedName, err := randomName(ext)
	if err != nil {
		return nil, apperror.Internal("gagal membuat nama file", err)
	}

	fullPath := filepath.Join(s.baseDir, storedName)
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, apperror.Internal("gagal menyimpan file", err)
	}
	defer dst.Close()

	limit := s.maxBytes
	if limit <= 0 {
		limit = upload.Size
	}
	written, err := io.Copy(dst, io.LimitReader(upload.Content, limit+1))
	if err != nil {
		_ = os.Remove(fullPath)
		return nil, apperror.Internal("gagal menulis file", err)
	}
	if s.maxBytes > 0 && written > s.maxBytes {
		_ = os.Remove(fullPath)
		return nil, apperror.Validation("file terlalu besar", map[string]string{
			"file": fmt.Sprintf("maksimal %d MB", s.maxBytes/(1024*1024)),
		})
	}

	return &domainstorage.Stored{
		FileName: originalName,
		Path:     filepath.ToSlash(filepath.Join(s.baseDir, storedName)),
	}, nil
}

func (s *localStorage) Open(ctx context.Context, path string) (*domainstorage.Content, error) {
	if err := ctx.Err(); err != nil {
		return nil, apperror.Internal("permintaan dibatalkan", err)
	}

	safePath, err := s.resolve(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(safePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, apperror.NotFound("file tidak ditemukan di penyimpanan")
		}
		return nil, apperror.Internal("gagal membuka file", err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, apperror.Internal("gagal membaca info file", err)
	}

	return &domainstorage.Content{Reader: file, Size: info.Size()}, nil
}

func (s *localStorage) Delete(ctx context.Context, path string) error {
	safePath, err := s.resolve(path)
	if err != nil {
		return err
	}
	if err := os.Remove(safePath); err != nil && !os.IsNotExist(err) {
		log.Printf("[WARN] gagal menghapus file %q: %v", safePath, err)
		return apperror.Internal("gagal menghapus file", err)
	}
	return nil
}

func (s *localStorage) resolve(path string) (string, error) {
	base, err := filepath.Abs(s.baseDir)
	if err != nil {
		return "", apperror.Internal("path upload tidak valid", err)
	}
	target, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", apperror.Internal("path file tidak valid", err)
	}
	rel, err := filepath.Rel(base, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", apperror.Validation("path file tidak valid", nil)
	}
	return target, nil
}

func randomName(ext string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ext, nil
}
