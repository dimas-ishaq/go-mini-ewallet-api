package main

// import: daftar paket yang dipakai program.
// "go-mini-ewallet-api/pkg/database" -> kode proyek sendiri untuk koneksi Postgres.
// "log" -> menulis pesan dan error ke console.
// "os" -> membaca environment variable seperti DB_HOST dan PORT.
// gin dan godotenv adalah dependency pihak ketiga (tercantum di go.mod).
import (
	"log"
	"os"

	handler "go-mini-ewallet-api/internal/delivery/http"
	"go-mini-ewallet-api/internal/repository"
	"go-mini-ewallet-api/internal/usecase"
	"go-mini-ewallet-api/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Tujuan   : main adalah entry point aplikasi Go, yaitu fungsi pertama yang
//
//	dijalankan saat program dimulai.
//
// Penggunaan: dipanggil otomatis oleh Go runtime; tidak dipanggil manual.
// Parameter : tidak ada (func main()).
// Return   : tidak ada (func main()).
func main() {
	// Baca key=value dari file .env ke environment proses.
	// Jika file tidak ada, program tetap lanjut; environment bisa disetel dari OS.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Ambil setting koneksi dari environment, lalu kumpulkan dalam Config.
	dbConfig := database.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	// Buat pool koneksi database. Jika gagal, aplikasi tidak bisa bekerja,
	// jadi log.Fatalf menulis error lalu menghentikan program.
	db, err := database.NewPostgresConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// defer menjalankan db.Close() saat main selesai, termasuk saat fungsi return.
	defer db.Close()
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Gagal migrasi database: %v", err)
	}

	// ERROR DI BARIS 56-57: nama constructor yang dipanggil tidak ada di package repository.
	// Constructor yang tersedia bernama NewPostgresUserRepository dan NewPostgresWalletRepository.
	// Menggunakan nama yang belum dideklarasikan memicu compile error `undefined`.
	userRepo := repository.NewPostgresUserRepository(db)
	walletRepo := repository.NewPostgresWalletRepository(db)

	// initialize usecase
	userUsecase := usecase.NewUserUsecase(userRepo, walletRepo)
	walletUsecase := usecase.NewWalletUsecase(walletRepo)

	// initialize handler
	userHandler := handler.NewUserHandler(userUsecase)
	walletHandler := handler.NewWalletHandler(walletUsecase)

	// gin.Default membuat router HTTP dengan middleware logger dan recovery.
	r := gin.Default()

	// ERROR SINTAKS:
	// 1. Baris `api:=r.Group("/api/v1"){\n...}` salah karena Go tidak mendukung block syntax `{}` setelah deklarasi fungsi.
	// 2. `userHandler.Register()` dan `userHandler.Login()` salah karena memanggil method bukan me-pass method value.
	// 3. `api.POST(("/login", ...))` kelebihan tanda kurung buka.
	// 4. `api.Group("").use(...)` salah huruf kapital (harus `Use`) dan diikuti block syntax `{}`.
	// 5. `walletHandler.GetWallets()` dan `walletHandler.CreateWallet()` tidak ada di `WalletHandler`.
	//    Nama method yang benar di wallet_handler.go: `GetBalance` dan `Transfer`.
	api := r.Group("/api/v1")
	{
		// public routes
		api.POST("/register", userHandler.Register)
		api.POST("/login", userHandler.Login)

		protected := api.Group("")
		protected.Use(handler.AuthMiddleware())
		{
			protected.GET("/wallet/balance", walletHandler.GetBalance)
			protected.POST("/wallet/transfer", walletHandler.Transfer)
		}
	}

	// Daftarkan endpoint GET /health untuk cek aplikasi hidup.
	// c adalah konteks request/response milik Gin.
	r.GET("/api/health", func(c *gin.Context) {
		// Balas HTTP 200 dengan JSON. gin.H adalah map[string]interface{}.
		c.JSON(200, gin.H{
			"status":   "OK",
			"database": "Connected",
		})
	})

	// Baca port dari environment; jika kosong, gunakan 8080.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s...", port)
	// Jalankan HTTP server. ":" + port jadi alamat seperti ":8080".
	// Jika server gagal start, hentikan program dan tampilkan penyebab.
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
