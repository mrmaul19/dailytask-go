-- Skema tabel users.
CREATE TABLE
    IF NOT EXISTS users (
        id BIGSERIAL PRIMARY KEY,
        email VARCHAR(100) NOT NULL DEFAULT '',
        password VARCHAR(100) NOT NULL DEFAULT '',
        name VARCHAR(100) NOT NULL DEFAULT '',
        address VARCHAR(255) NOT NULL DEFAULT '',
        job VARCHAR(100) NOT NULL DEFAULT '',
        status VARCHAR(100) NOT NULL DEFAULT '',
        created_at TIMESTAMPTZ NOT NULL DEFAULT now (),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT now (),
        deleted_at TIMESTAMPTZ
    );

-- Semua query membawa "deleted_at IS NULL", jadi kolom ini layak diindeks.
CREATE INDEX IF NOT EXISTS users_deleted_at_idx ON users (deleted_at);