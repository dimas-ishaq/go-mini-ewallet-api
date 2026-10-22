package repository

// import: paket yang dipakai file ini.
// "context"          -> context.Context untuk timeout/cancel query DB.
// "database/sql"     -> sql.DB, QueryRowContext, ErrNoRows (API database standar Go).
// "fmt"              -> fmt.Errorf untuk membungkus error dengan pesan.
// "go-mini-.../domain" -> tipe domain.User dan interface domain.UserRepository dari proyek sendiri.
import (
	"context"
	"database/sql"
	"fmt"
	"go-mini-ewallet-api/internal/domain"
)

// postgresUserRepository pelaksana interface domain.UserRepository khusus Postgres.
// Menyimpan *sql.DB (pool koneksi) agar bisa query ke database.
// Huruf kecil di awal nama = tidak bisa diakses dari luar paket (private).
type postgresUserRepository struct {
	db *sql.DB // pool koneksi yang dibuat di pkg/database/postgres.go
}

// Tujuan   : buat repository Postgres yang memenuhi kontrak domain.UserRepository.
// Penggunaan: panggil saat merangkai aplikasi, lalu berikan hasilnya ke service.
//            repo := NewPostgresUserRepository(db)
// Parameter : db *sql.DB adalah pool koneksi Postgres yang sudah dibuka.
// Return   : domain.UserRepository (interface) yang bisa dipakai untuk Create/GetByID/GetByEmail.
func NewPostgresUserRepository(db *sql.DB) domain.UserRepository {
	return &postgresUserRepository{db: db}
}

// Tujuan   : menyimpan user baru ke tabel users.
// Penggunaan: panggil setelah data user divalidasi dan PasswordHash sudah dibuat.
// Parameter : ctx context.Context untuk timeout/cancel query; user *domain.User
//            berisi data yang disimpan. Pointer agar ID dan CreatedAt hasil DB
//            bisa ditulis kembali ke struct yang sama.
// Return   : error nil jika berhasil; error berisi penyebab jika query gagal.
func (r *postgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	// $1,$2,$3 adalah placeholder Postgres (bukan "?") agar aman dari SQL injection.
	// RETURNING id, created_at = minta DB kembalikan nilai yang baru dibuat (auto-increment & timestamp).
	query := `
	INSERT INTO users (name, email,password_hash) VALUES ($1,$2,$3) RETURNING id, created_at
	`
	// QueryRowContext = kirim 1 baris query + tunggu 1 baris hasil.
	// Scan(&user.ID, &user.CreatedAt) = salin kolom hasil ke field struct lewat pointer.
	err := r.db.QueryRowContext(ctx, query, user.Name, user.Email, user.PasswordHash).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		// %w = bungkus error asli agar stack penyebab tetap terbaca di log.
		return fmt.Errorf("gagal insert user: %w", err)
	}
	return nil
}

// Tujuan   : mencari satu user berdasarkan ID-nya.
// Penggunaan: panggil saat perlu mengambil profil user; repo.GetByID(ctx, 42).
// Parameter : ctx untuk timeout/cancel query; id int adalah ID user yang dicari.
// Return   : (*domain.User, nil) jika ditemukan; (nil, nil) jika tidak ada;
//            (nil, error) jika database/query bermasalah.
func (r *postgresUserRepository) GetByID(ctx context.Context, id int) (*domain.User, error) {
	// Cari satu baris berdasarkan ID; placeholder $1 menjaga nilai tetap sebagai parameter query.
	query := `SELECT id, name, email, password_hash, created_at FROM users WHERE id = $1`

	var user domain.User // wadah kosong untuk menampung hasil Scan
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
 // sql.ErrNoRows berarti ID tidak cocok dengan user; error lain menandakan query gagal.
		if err == sql.ErrNoRows {
			return nil, nil // User tidak ditemukan -> bukan error, hanya kosong
		}
		return nil, fmt.Errorf("gagal get user by id: %w", err)
	}

	return &user, nil // &user = kirim alamat wadah yang sudah terisi
}

// Tujuan   : mencari satu user berdasarkan email. Umumnya untuk proses login.
// Penggunaan: user, err := repo.GetByEmail(ctx, "budi@mail.com") lalu bandingkan password.
// Parameter : ctx untuk timeout/cancel query; email string adalah email yang dicari.
// Return   : (*domain.User, nil) jika ditemukan; (nil, nil) jika tidak ada;
//            (nil, error) jika database/query bermasalah.
func (r *postgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
 // Cari satu baris berdasarkan email dengan parameter terikat, bukan gabungan string SQL.
	query := `SELECT id, name, email, password_hash, created_at FROM users WHERE email = $1`

	var user domain.User
	err := r.db.QueryRowContext(ctx, query, email).
		Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
 // sql.ErrNoRows berarti email belum terdaftar; error lain menandakan query gagal.
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal get user by email: %w", err)
	}

 // Hasil Scan mengisi field user; alamatnya dikembalikan agar pemanggil menerima record.
 // Kembalikan user yang telah diisi oleh Scan.
	return &user, nil
}
