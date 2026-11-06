-- Menambahkan email & password ke tabel users.
--
-- Kenapa file baru, bukan mengedit 0003_create_users.sql?
-- Runner migrasi mencatat riwayat PER NAMA FILE di tabel schema_migrations.
-- File 0003 sudah pernah dieksekusi, jadi perubahan apa pun di dalamnya tidak
-- akan pernah dijalankan ulang: database yang sudah ada tidak akan mendapat
-- kolom barunya, tanpa satu pun pesan error.
--
-- Semua perintah di bawah idempoten, jadi aman untuk dua kondisi sekaligus:
--   - database lama  -> kolomnya ditambahkan di sini;
--   - database baru  -> kolomnya sudah dibuat 0003, perintah ini jadi no-op.

ALTER TABLE users ADD COLUMN IF NOT EXISTS email    VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS password VARCHAR(255) NOT NULL DEFAULT '';

-- Menyamakan lebar kolom untuk database yang tabelnya dibuat sebelum perubahan
-- ini, supaya skema database lama dan baru benar-benar identik.
-- Hash bcrypt panjangnya 60 karakter; 255 memberi ruang kalau algoritmanya
-- nanti diganti (argon2id menghasilkan string yang lebih panjang).
ALTER TABLE users ALTER COLUMN email    TYPE VARCHAR(255);
ALTER TABLE users ALTER COLUMN password TYPE VARCHAR(255);

-- Email wajib unik, tetapi:
--   - lower(email) -> perbandingan case-insensitive, sejalan dengan
--     entity.NormalizeEmail;
--   - WHERE deleted_at IS NULL -> email bekas user yang dihapus boleh dipakai lagi;
--   - WHERE email <> '' -> baris lama yang email-nya masih kosong (hasil DEFAULT '')
--     tidak saling bentrok.
CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_idx
    ON users (lower(email))
    WHERE deleted_at IS NULL AND email <> '';
