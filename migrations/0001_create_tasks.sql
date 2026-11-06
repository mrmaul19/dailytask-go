-- Skema awal tabel tasks.
--
-- Kolom teks dibuat NOT NULL DEFAULT '' supaya bisa di-Scan langsung ke
-- string biasa di Go, tanpa perlu sql.NullString di layer repository.
-- deleted_at dibiarkan nullable karena dipakai untuk soft delete:
-- baris yang "dihapus" hanya diisi timestamp, bukan dibuang.
CREATE TABLE IF NOT EXISTS tasks (
    id          BIGSERIAL     PRIMARY KEY,
    no_task     VARCHAR(50)   NOT NULL,
    judul       VARCHAR(255)  NOT NULL,
    description TEXT          NOT NULL DEFAULT '',
    file_name   VARCHAR(255)  NOT NULL DEFAULT '',
    file_path   VARCHAR(512)  NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

-- Unik hanya di antara baris yang belum di-soft-delete, jadi nomor task
-- bisa dipakai ulang setelah datanya dihapus.
CREATE UNIQUE INDEX IF NOT EXISTS tasks_no_task_active_unique
    ON tasks (no_task) WHERE deleted_at IS NULL;

-- Semua query membawa "deleted_at IS NULL", jadi kolom ini layak diindeks.
CREATE INDEX IF NOT EXISTS tasks_deleted_at_idx ON tasks (deleted_at);
