package database

// import: paket yang dipakai file ini.
// "database/sql" -> tipe sql.DB, API database generik Go (tidak terikat Postgres saja).
// "fmt"          -> fmt.Sprintf untuk merangkai string koneksi (DSN), fmt.Errorf untuk error berkonteks.
// "log"          -> log.Println untuk pesan ke console.
// _ "github.com/lib/pq" -> driver Postgres. Garis bawah "_" = blank import:
//   paket diimpor hanya agar fungsi init()-nya jalan (mendaftarkan driver "postgres"
//   ke database/sql), tanpa dipakai langsung di kode. Tanpa ini sql.Open("postgres") gagal.
import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

// Config wadah setting koneksi database.
// Diisi dari environment variable di main.go (DB_HOST, DB_PORT, dst).
// Dipisah jadi struct agar gampang di-test dan tidak oper banyak parameter.
type Config struct {
	// Host alamat server DB (contoh: "localhost").
	Host string
	// Port port Postgres (umumnya "5432").
	Port string
	// User nama user database.
	User string
	// Password password user database.
	Password string
	// DBName nama database yang dipakai.
	DBName string
	// SSLMode mode enkripsi koneksi ("disable" untuk lokal, "require" untuk production).
	SSLMode string
}

// Tujuan   : buka koneksi (pool) ke Postgres dan pastikan siap dipakai.
// Penggunaan: dipanggil sekali saat aplikasi start.
//            c := database.Config{...}; db, err := database.NewPostgresConnection(c)
//            jika err != nil -> log & hentikan start.
// Parameter : cfg Config berisi Host/Port/User/Password/DBName/SSLMode.
// Return   : (*sql.DB, error). sukses -> (pool koneksi, nil); gagal -> (nil, error terbungkus fmt.Errorf).
func NewPostgresConnection(cfg Config) (*sql.DB, error) {
	// DSN (Data Source Name): string koneksi "kunci=nilai dipisah spasi".
	// Tiap %s adalah placeholder untuk satu string; urutannya HARUS sama
	// dengan argumen setelah koma: Host->%s pertama, Port->%s kedua, dst.
	// Hasil contoh: "host=localhost port=5432 user=postgres password=xxx dbname=ewallet sslmode=disable".
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	// sql.Open TIDAK langsung konek; ia hanya siapkan pool + cek argumen driver.
	// "postgres" adalah nama driver yang didaftarkan oleh lib/pq lewat blank import.
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		// %w (atau %v) membungkus error asli + pesan kita, agar penyebabnya terlacak.
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}

 // Ping memvalidasi koneksi nyata setelah sql.Open menyiapkan pool; kegagalan
 // dikembalikan agar pemanggil dapat menghentikan startup sebelum menerima request.
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// Batasi pool agar DB tidak kebanjiran koneksi.
	// SetMaxOpenConns(25): maksimal 25 koneksi bersamaan.
	db.SetMaxOpenConns(25)
	// SetMaxIdleConns(5): maksimal 5 koneksi nganggur yang disimpan untuk dipakai ulang.
	db.SetMaxIdleConns(5)
	log.Println("Berhasil terhubung ke database PostgreSQL")
	return db, nil
}
