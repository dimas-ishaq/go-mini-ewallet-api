package domain

// import digunakan untuk memasukkan paket lain ke file ini.
// Tanpa import, kita tidak bisa pakai tipe/fungsi dari paket tersebut.
// Nama di dalam tanda kutip = path paket: untuk stdlib cuma nama paket
// (mis. "context", "time"), untuk paket luar bisa seperti "github.com/...".
// Catatan: Go akan menolak compile jika ada import yang tidak terpakai.
import (
	// "context" memberi tipe context.Context.
	// Digunakan di parameter UserRepository (Create/GetByID/GetByEmail)
	// untuk mengatur timeout & pembatalan (cancel) saat akses database.
	"context"
	// "time" memberi tipe time.Time.
	// Digunakan pada field User.CreatedAt untuk menyimpan waktu pembuatan.
	"time"
)

// User adalah data pengguna.
// Satu struct = satu baris di tabel users.
type User struct {
	// ID nomor unik pengguna dari database (auto-increment).
	ID int `json:"id"`
	// Name nama tampilan pengguna.
	Name string `json:"name"`
	// Email alamat unik untuk login. Dipakai di GetByEmail.
	Email string `json:"email"`
	// PasswordHash hasil hash password, bukan password asli.
	// json:"-" artinya field ini TIDAK PERNAH dikirim ke JSON (keamanan).
	PasswordHash string `json:"-"`
	// CreatedAt waktu baris dibuat. Diisi database / saat Create.
	CreatedAt time.Time `json:"created_at"`
}

// UserRepository kontrak akses data user.
// Ini interface: hanya daftar fungsi, tanpa isi.
// Pelaksana aslinya ada di lapisan repository (misal postgres).
// Kode service cukup panggil interface ini, tidak perlu tahu SQL-nya.
type UserRepository interface {
	// Tujuan   : simpan user baru.
	// Penggunaan: panggil setelah user divalidasi dan password di-hash.
	// Parameter : ctx membatasi/membatalkan operasi; user berisi data baru.
	// Return   : nil jika berhasil; error jika penyimpanan gagal.
	Create(ctx context.Context, user *User) error
	// Tujuan   : ambil user menggunakan ID.
	// Penggunaan: panggil saat perlu mencari profil user.
	// Parameter : ctx untuk cancel/timeout; id adalah ID yang dicari.
	// Return   : pointer user jika ditemukan; nil,nil jika tidak ditemukan;
	//            nil,error jika terjadi kegagalan database.
	GetByID(ctx context.Context, id int) (*User, error)
	// Tujuan   : ambil user menggunakan email, umumnya saat login.
	// Penggunaan: panggil untuk mencari akun sebelum memverifikasi password.
	// Parameter : ctx untuk cancel/timeout; email adalah alamat yang dicari.
	// Return   : pointer user jika ditemukan; nil,nil jika tidak ditemukan;
	//            nil,error jika terjadi kegagalan database.
	GetByEmail(ctx context.Context, email string) (*User, error)
}
