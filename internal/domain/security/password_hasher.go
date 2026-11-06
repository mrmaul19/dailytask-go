// Package security mendefinisikan port untuk pengamanan password.
//
// Sengaja dibuat sebagai interface di domain, bukan memanggil bcrypt langsung
// dari usecase, dengan dua alasan:
//
//  1. usecase tetap bebas dari library pihak ketiga — algoritmanya bisa diganti
//     (bcrypt -> argon2id) tanpa menyentuh satu baris pun aturan bisnis;
//  2. unit test bisa memakai hasher palsu yang cepat, karena bcrypt sengaja
//     lambat sehingga membuat test ikut lambat kalau dipakai langsung.
package security

// PasswordHasher mengubah password polos menjadi hash yang aman disimpan,
// dan memeriksa apakah sebuah password polos cocok dengan hash tersebut.
type PasswordHasher interface {
	// Hash mengembalikan hash dari password polos.
	// Implementasi wajib memakai salt acak, sehingga dua password yang sama
	// menghasilkan hash yang berbeda.
	Hash(plain string) (string, error)

	// Verify mengembalikan nil kalau plain cocok dengan hash.
	//
	// Pemanggil TIDAK boleh membedakan jenis error yang dikembalikan di sini
	// saat membalas client: "hash rusak" dan "password salah" harus menghasilkan
	// respons yang sama, supaya tidak ada informasi yang bocor.
	Verify(hash, plain string) error
}
