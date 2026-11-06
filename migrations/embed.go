// Package migrations menyimpan skema database sebagai file .sql biasa.
// File-nya di-embed ke dalam binary supaya aplikasi tetap bisa menjalankan
// migrasi tanpa perlu membawa folder migrations/ saat deploy.
//
// Aturan penamaan: <nomor urut>_<deskripsi>.sql, dijalankan urut nama file.
// File yang sudah pernah dijalankan TIDAK BOLEH diubah lagi -- buat file baru,
// karena riwayatnya dicatat per nama file di tabel schema_migrations.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
