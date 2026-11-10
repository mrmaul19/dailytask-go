package entity

import (
	"errors"
	"time"
)

var ErrTimeEntryNotFound = errors.New("catatan waktu tidak ditemukan")
var (
	ErrTimerSedangBerjalan    = errors.New("masih ada timer yang berjalan")
	ErrTimerTidakBerjalan     = errors.New("tidak ada timer yang sedang berjalan")
	ErrRentangWaktuTidakValid = errors.New("waktu selesai harus setelah waktu mulai")
)

type TimeEntry struct {
	ID        uint
	UserID    uint
	TaskID    uint
	StartedAt time.Time
	EndedAt   *time.Time
	Note      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (e *TimeEntry) IsRunning() bool { return e.EndedAt == nil }

func (e *TimeEntry) Stop(at time.Time) error {
	if !e.IsRunning() {
		return ErrTimerTidakBerjalan
	}
	if !at.After(e.StartedAt) {
		return ErrRentangWaktuTidakValid
	}

	e.EndedAt = &at
	return nil
}

func (e *TimeEntry) Validate() error {
	if e.IsRunning() {
		return nil
	}
	if !e.EndedAt.After(e.StartedAt) {
		return ErrRentangWaktuTidakValid
	}
	return nil
}

func (e *TimeEntry) DurationAt(now time.Time) time.Duration {
	if e.IsRunning() {
		return now.Sub(e.StartedAt)
	}
	return e.EndedAt.Sub(e.StartedAt)
}

func (e *TimeEntry) Overlaps(other *TimeEntry) bool {
	if e == nil || other == nil {
		return false
	}

	mulaiSebelumAkhir := func(mulai time.Time, lawan *TimeEntry) bool {
		return lawan.IsRunning() || mulai.Before(*lawan.EndedAt)
	}

	return mulaiSebelumAkhir(e.StartedAt, other) && mulaiSebelumAkhir(other.StartedAt, e)
}
