CREATE TABLE
    IF NOT EXISTS time_entries (
        id BIGSERIAL PRIMARY KEY,
        user_id BIGINT NOT NULL REFERENCES users (id),
        task_id BIGINT NOT NULL REFERENCES tasks (id),
        started_at TIMESTAMPTZ NOT NULL,
        ended_at TIMESTAMPTZ,
        note VARCHAR(255) NOT NULL DEFAULT '',
        created_at TIMESTAMPTZ NOT NULL DEFAULT now (),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT now (),
        deleted_at TIMESTAMPTZ,
        CONSTRAINT time_entries_interval_check CHECK (
            ended_at IS NULL
            OR ended_at > started_at
        )
    );

CREATE UNIQUE INDEX IF NOT EXISTS time_entries_one_running_per_user ON time_entries (user_id)
WHERE
    ended_at IS NULL
    AND deleted_at IS NULL;

-- Untuk "riwayat waktu pada satu task", diurutkan dari yang terbaru.
CREATE INDEX IF NOT EXISTS time_entries_task_started_idx ON time_entries (task_id, started_at DESC)
WHERE
    deleted_at IS NULL;

-- Untuk "riwayat waktu milik satu user" dan laporan rentang tanggal.
CREATE INDEX IF NOT EXISTS time_entries_user_started_idx ON time_entries (user_id, started_at DESC)
WHERE
    deleted_at IS NULL;
