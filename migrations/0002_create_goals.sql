-- Skema tabel goals.
--
-- Pola kolomnya sengaja sama dengan tabel tasks: kolom teks NOT NULL DEFAULT ''
-- supaya bisa di-Scan langsung ke string biasa, dan deleted_at nullable
-- untuk soft delete.
CREATE TABLE IF NOT EXISTS goals (
    id         BIGSERIAL    PRIMARY KEY,
    title      VARCHAR(255) NOT NULL,
    category   VARCHAR(100) NOT NULL DEFAULT '',
    priority   VARCHAR(100) NOT NULL DEFAULT '',
    status     VARCHAR(100) NOT NULL DEFAULT '',
    progress   VARCHAR(100) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- Semua query membawa "deleted_at IS NULL", jadi kolom ini layak diindeks.
CREATE INDEX IF NOT EXISTS goals_deleted_at_idx ON goals (deleted_at);
