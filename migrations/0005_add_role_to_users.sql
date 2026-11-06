-- Menambahkan role ke tabel users, untuk membedakan admin dari user biasa.
--
-- File baru, bukan mengedit 0003/0004 — riwayat migrasi dicatat per nama file,
-- jadi file yang sudah pernah jalan tidak akan pernah dieksekusi ulang.

-- DEFAULT 'user' dipilih dengan sengaja: semua baris yang sudah ada otomatis
-- menjadi user biasa. Kebalikannya (DEFAULT 'admin') akan langsung memberi hak
-- penuh kepada setiap akun yang terlanjur ada di database.
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user';

-- Lapis pertahanan terakhir kalau ada kode (atau tangan manusia di psql) yang
-- menulis role di luar dua nilai yang dikenal. Aplikasi sudah memvalidasi di
-- usecase, tetapi database tidak boleh bergantung pada itikad baik pemanggilnya.
--
-- ADD CONSTRAINT tidak punya bentuk IF NOT EXISTS, jadi keberadaannya diperiksa
-- manual supaya migrasi ini tetap aman dijalankan ulang.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'users_role_check'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT users_role_check CHECK (role IN ('user', 'admin'));
    END IF;
END
$$;

-- Admin pertama tidak bisa lahir dari endpoint mana pun: /auth/register selalu
-- membuat role 'user', dan /api/v1/users sendiri sudah butuh token admin.
-- Jadi admin pertama dipromosikan manual, sekali seumur hidup database:
--
--     UPDATE users SET role = 'admin' WHERE email = 'email-anda@contoh.com';
--
-- Sengaja TIDAK dilakukan otomatis di sini, karena itu berarti menanam
-- hash password ke dalam file migrasi yang ikut masuk ke repo selamanya.
