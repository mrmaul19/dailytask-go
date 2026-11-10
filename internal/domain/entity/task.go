package entity

import (
	"errors"
	"strings"
	"time"
)

var ErrTaskNotFound = errors.New("task tidak ditemukan")

type Task struct {
	ID          uint
	NoTask      string
	Judul       string
	Description string
	FileName    string
	FilePath    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (t *Task) HasFile() bool { return strings.TrimSpace(t.FilePath) != "" }

func (t *Task) AttachFile(name, path string) {
	t.FileName = name
	t.FilePath = path
}

func (t *Task) DetachFile() {
	t.FileName = ""
	t.FilePath = ""
}
