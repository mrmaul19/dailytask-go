// Package entity berisi model inti bisnis. Lapisan ini paling dalam:
// TIDAK boleh mengimpor gorm, fiber, atau paket infrastruktur apa pun.
package entity

import (
	"errors"
	"strings"
	"time"
)

var ErrTaskNotFound = errors.New("task tidak ditemukan")

// Task merepresentasikan satu daily task.
type Task struct {
	ID          uint
	NoTask      string // nomor/kode task, unik. contoh: TASK-20261102-A1B2
	Judul       string
	Description string
	FileName    string // nama asli file yang diupload
	FilePath    string // path relatif di penyimpanan, contoh: uploads/xxx.pdf
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (t *Task) HasFile() bool { return strings.TrimSpace(t.FilePath) != "" }

// AttachFile menempelkan informasi file hasil upload ke task.
func (t *Task) AttachFile(name, path string) {
	t.FileName = name
	t.FilePath = path
}

func (t *Task) DetachFile() {
	t.FileName = ""
	t.FilePath = ""
}
