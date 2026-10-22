package usecase

// import: daftar pustaka yang digunakan file ini.
// "context"                 -> untuk membatasi durasi eksekusi (timeout/cancel).
// "crypto/rand"             -> sumber acak kriptografis untuk nomor rekening.
// "errors"                  -> membuat error sentinel (variabel error untuk errors.Is).
// "fmt"                     -> format string dan bungkus error (%w).
// "math/big"                -> bilangan besar untuk menghasilkan nomor rekening 8 digit.
// "strings"                 -> TrimSpace/ToLower untuk normalisasi nama & email.
// "sync"                    -> sync.OnceValue agar hash bcrypt dummy dibuat hanya sekali.
// "golang.org/x/crypto/bcrypt" -> enkripsi satu arah (hash) dan verifikasi password.
// "go-mini-ewallet-api/internal/domain" -> model User, Wallet, dan kontrak repository.
import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"go-mini-ewallet-api/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

// Error sentinel: variabel error yang diekspor agar lapisan handler cukup
// menulis errors.Is(err, usecase.ErrX) tanpa cocokkan string pesan.
var (
	// ErrValidation = fondasi untuk semua kesalahan validasi input.
	// Pesan specifics selalu dibungkus dengan %w dari sentinel ini.
	ErrValidation = errors.New("validasi input gagal")
	// ErrEmailRegistered = email sudah dipakai akun lain.
	ErrEmailRegistered = errors.New("email sudah terdaftar")
	// ErrInvalidCredentials = login gagal. Satu pesan yang sama untuk email
	// tidak ada maupun password salah, supaya tidak membocorkan keberadaan akun.
	ErrInvalidCredentials = errors.New("email atau password salah")
	// ErrAccountNumberExhausted = nomor rekening acak bertabrakan terus
	// hingga batas percobaan, sehingga registrasi dibatalkan (fail-safe).
	ErrAccountNumberExhausted = errors.New("gagal membuat nomor rekening, silakan coba lagi")
)

// Batas validasi input (boundary). Nilai sengaja dibuat eksplisit agar
// aturan bisnis tidak tersebar di beberapa lapisan.
const (
	minPasswordLength = 8
	// maxPasswordLength = 72 byte. Batas keras algoritma bcrypt: input yang
	// lebih panjang DIHALANGKAN, bukan dipotong. Jika dipotong, dua password
	// yang berbeda hanya pada bagian setelah byte ke-72 akan menghasilkan
	// hash yang sama, sehingga siapa pun bisa login memakai password latter.
	maxPasswordLength = 72
	maxNameLength     = 100
	maxEmailLength    = 254 // RFC 5321 batas panjang alamat email
	maxLocalPartLen   = 64
)

// Parameter pembentuk nomor rekening: prefix "10" + 8 digit angka acak.
const (
	accountNumberPrefix  = "10"
	accountNumberDigits  = 8
	accountNumberSpace   = 100000000 // 10^8, batas atas eksklusif untuk angka acak
	maxAccountGenAttempt = 10        // batas percobaan ulang bila angka sudah terpakai
)

// dummyPasswordHash menghasilkan sebuah hash bcrypt valid, satu kali saja.
// Dipakai di Login saat email tidak ditemukan: perbandingan bcrypt tetap
// dijalankan agar waktu respons identik dengan kasus password salah.
// Tanpa ini, penyerang bisa menebak email mana yang terdaftar dari selisih
// waktu respons (user enumeration via timing attack).
var dummyPasswordHash = sync.OnceValue(func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing-only"), bcrypt.DefaultCost)
	if err != nil {
		// bcrypt hanya gagal bila cost di luar rentang; DefaultCost selalu valid.
		panic(fmt.Sprintf("gagal membuat dummy password hash: %v", err))
	}
	return hash
})

// isValidEmail melakukan pemeriksaan bentuk alamat email yang wajar.
// Cukup untuk keperluan registrasi (bukan parser RFC 5322 lengkap).
func isValidEmail(email string) bool {
	// Periksa batas maksimum panjang string email serta tolak karakter whitespace/spasi.
	if len(email) > maxEmailLength || strings.ContainsAny(email, " \t\r\n") {
		return false
	}
	// Cari posisi karakter pemisah '@' terakhir dalam string email.
	at := strings.LastIndex(email, "@")
	// Karakter '@' tidak boleh berada di awal indeks (0) maupun di karakter paling akhir.
	if at <= 0 || at == len(email)-1 {
		return false
	}
	// Pecah alamat email menjadi bagian lokal (nama user) dan bagian nama domain.
	local, domainPart := email[:at], email[at+1:]
	// Pastikan bagian lokal tidak melampaui batas maksimal panjang karakter yang diizinkan.
	if len(local) > maxLocalPartLen {
		return false
	}
	// Domain wajib punya titik, tidak boleh diawali/diakhiri titik,
	// dan tidak boleh mengandung dua titik bertumpuk ("a..b").
	if !strings.Contains(domainPart, ".") ||
		strings.HasPrefix(domainPart, ".") ||
		strings.HasSuffix(domainPart, ".") ||
		strings.Contains(domainPart, "..") {
		return false
	}
	// Seluruh pengecekan sintaks email dasar terpenuhi.
	return true
}

// validateRegisterInput menormalkan lalu memvalidasi input registrasi.
// Normalisasi (trim + lowercase email) wajib dilakukan SEBELUM pengecekan
// ke database, dan dipakai ulang saat Login, supaya "Budi@Mail.com" dan
// "budi@mail.com" tidak menjadi dua akun berbeda.
func validateRegisterInput(name, email, password string) (string, string, error) {
	// Hilangkan spasi berlebih pada awal dan akhir string nama.
	name = strings.TrimSpace(name)
	// Ubah seluruh karakter email menjadi huruf kecil dan bersihkan spasi awal/akhir.
	email = strings.ToLower(strings.TrimSpace(email))

	// Validasi kelengkapan data: nama pengguna tidak boleh kosong.
	if name == "" {
		return "", "", fmt.Errorf("%w: nama wajib diisi", ErrValidation)
	}
	// Hitung jumlah rune/karakter UTF-8 nama dan pastikan tidak melebihi batas maksimum.
	if len([]rune(name)) > maxNameLength {
		return "", "", fmt.Errorf("%w: nama maksimal %d karakter", ErrValidation, maxNameLength)
	}
	// Validasi kelengkapan data: alamat email wajib diisi.
	if email == "" {
		return "", "", fmt.Errorf("%w: email wajib diisi", ErrValidation)
	}
	// Uji kesesuaian format alamat email dengan aturan format standar.
	if !isValidEmail(email) {
		return "", "", fmt.Errorf("%w: format email tidak valid", ErrValidation)
	}
	// Password sengaja tidak di-trim: spasi adalah bagian password yang sah.
	// Pastikan panjang string password memenuhi batas minimum keamanan.
	if len(password) < minPasswordLength {
		return "", "", fmt.Errorf("%w: password minimal %d karakter", ErrValidation, minPasswordLength)
	}
	// Batasi ukuran byte password agar tidak melebihi kapasitas hashing bcrypt.
	if len(password) > maxPasswordLength {
		return "", "", fmt.Errorf("%w: password maksimal %d byte", ErrValidation, maxPasswordLength)
	}
	// Seluruh data masukan valid, kembalikan nama dan email yang sudah dinormalisasi.
	return name, email, nil
}

// randomAccountNumber membuat nomor rekening 10 digit ("10" + 8 angka acak)
// memakai crypto/rand.
//
// PERBAIKAN: versi lama memakai math/rand + rand.Seed(time.Now().UnixNano())
// di dalam setiap pemanggilan. rand.Seed sudah deprecated sejak Go 1.20 dan
// karena menyetel sumber acak global, pemanggilan bersamaan bisa saling
// menabrak state dan menghasilkan nomor yang sama. Untuk nilai yang menentukan
// tujuan transfer uang, sumber acak kriptografis adalah pilihan yang tepat.
func randomAccountNumber() (string, error) {
	// rand.Int melakukan sampling uniform pada [0, accountNumberSpace).
	n, err := rand.Int(rand.Reader, big.NewInt(accountNumberSpace))
	if err != nil {
		return "", fmt.Errorf("gagal membuat angka acak: %w", err)
	}
	// %0*d mengisi nol di depan sampai panjang tepat accountNumberDigits.
	return accountNumberPrefix + fmt.Sprintf("%0*d", accountNumberDigits, n.Int64()), nil
}

// generateAccountNumber menghasilkan nomor rekening yang belum dipakai user lain.
//
// PERBAIKAN: versi lama sama sekali tidak mengecek keunikan, sehingga dua
// user bisa mendapat nomor rekening identik dan uang transfer salah tujuan.
// Sekarang nomor acak dicoba sampai bebas atau menyentuh batas percobaan.
//
// CATATAN: pemeriksaan ini tetap berupa "read-then-write" dan masih raced
// jika ada dua registrasi yang benar-benar bersamaan. Jaring pengaman
// sesungguhnya harus berupa constraint UNIQUE pada kolom wallets.account_number
// di database; pemeriksa ini hanya menurunkan peluang tabrakan.
func (u *userUsecase) generateAccountNumber(ctx context.Context) (string, error) {
	for attempt := 0; attempt < maxAccountGenAttempt; attempt++ {
		accountNumber, err := randomAccountNumber()
		if err != nil {
			return "", err
		}
		existing, err := u.walletRepo.GetByAccountNumber(ctx, accountNumber)
		if err != nil {
			return "", fmt.Errorf("gagal cek nomor rekening: %w", err)
		}
		// GetByAccountNumber mengembalikan (nil, nil) saat tidak ditemukan.
		if existing == nil {
			return accountNumber, nil
		}
	}
	return "", ErrAccountNumberExhausted
}

// UserUsecase adalah kontrak (interface) logika bisnis untuk pengguna.
// Lapisan HTTP Handler hanya memanggil interface ini tanpa tahu detail database.
type UserUsecase interface {
	// Register membuat akun user baru sekaligus membuatkan dompet default.
	Register(ctx context.Context, name, email, password string) (*domain.User, *domain.Wallet, error)
	// Login mencocokkan email dan password, mengembalikan user jika valid.
	Login(ctx context.Context, email, password string) (*domain.User, error)
}

// userUsecase adalah struct yang menjalankan operasi bisnis sesungguhnya.
// Struct ini butuh userRepo dan walletRepo untuk berinteraksi dengan database.
type userUsecase struct {
	userRepo   domain.UserRepository
	walletRepo domain.WalletRepository
}

// Tujuan   : membuat instance UserUsecase baru (constructor).
// Penggunaan: dipanggil di main.go saat merakit ketergantungan aplikasi.
// Parameter : uRepo (akses tabel users), wRepo (akses tabel wallets).
// Return   : UserUsecase interface siap pakai.
func NewUserUsecase(uRepo domain.UserRepository, wRepo domain.WalletRepository) UserUsecase {
	return &userUsecase{
		userRepo:   uRepo,
		walletRepo: wRepo,
	}
}

// Tujuan   : mendaftarkan user baru dan langsung membuatkan wallet awal untuk user tersebut.
// Penggunaan: dipanggil saat endpoint HTTP POST /api/register menerima request.
// Parameter :
//   - ctx context.Context: kontrol pembatalan/timeout dari HTTP request.
//   - name string        : nama lengkap pengguna.
//   - email string       : email unik pengguna untuk login.
//   - password string    : password mentah dari client (belum di-hash).
//
// Return   :
//   - *domain.User  : pointer data user yang baru tersimpan (berisi ID baru).
//   - *domain.Wallet: pointer data wallet yang baru dibuatkan.
//   - error         : bernilai nil jika sukses; berisi pesan error jika gagal.
//
// Kenapa return memiliki 3 nilai (*domain.User, *domain.Wallet, error)?
// Karena Go mendukung multiple return values. Di use case registrasi e-wallet,
// kita ingin mengembalikan data user sekaligus wallet miliknya ke handler,
// serta status error. Bila terjadi kegagalan di tengah jalan, kembalikan (nil, nil, err).
func (u *userUsecase) Register(ctx context.Context, name, email, password string) (*domain.User, *domain.Wallet, error) {
	// TAHAP 0: Normalisasi + validasi input (boundary).
	// Dulu tahap ini tidak ada sama sekali, sehingga email kosong, password
	// kosong, atau password > 72 byte tetap berhasil dibuatkan akunnya.
	name, email, err := validateRegisterInput(name, email, password)
	if err != nil {
		return nil, nil, err
	}

	// TAHAP 1: Cek apakah email sudah pernah terdaftar di sistem.
	// Catatan: ini hanya pengecekan "read-then-write" sehingga tetap raced.
	// Jaring pengaman sesungguhnya adalah constraint UNIQUE pada kolom
	// users.email di database.
	existingUser, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Gagal query ke database (bukan karena data kosong).
		return nil, nil, fmt.Errorf("gagal cek email: %w", err)
	}
	if existingUser != nil {
		// Jika existingUser tidak nil, berarti email sudah dipakai akun lain.
		return nil, nil, ErrEmailRegistered
	}

	// TAHAP 2: Enkripsi (hash) password mentah menggunakan bcrypt.
	// Jangan pernah simpan password teks biasa (plain text) ke database.
	// DefaultCost adalah tingkat kesulitan standar algoritma bcrypt.
	// Panjang password sudah dijaga <= 72 byte oleh validateRegisterInput,
	// sehingga tidak ada risiko pemotonan diam-diam oleh bcrypt.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("gagal hash password: %w", err)
	}

	// TAHAP 3: Siapkan struct User dan simpan ke database melalui userRepo.
	user := &domain.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hashedPassword),
	}
	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, nil, fmt.Errorf("gagal menyimpan data user: %w", err)
	}

	// TAHAP 4: Buat nomor rekening acak untuk wallet baru user.
	// Nomor diambil dari crypto/rand lalu diverifikasi belum dipakai user lain,
	// sehingga tidak mungkin dua user memiliki nomor rekening yang sama
	// (versi lama tidak mengecek apa pun sehingga transfer bisa salah tujuan).
	accountNumber, err := u.generateAccountNumber(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("gagal membuat nomor rekening: %w", err)
	}

	// TAHAP 5: Siapkan struct Wallet dengan saldo awal 0 dan hubungkan ke user.ID.
	wallet := &domain.Wallet{
		UserID:        user.ID,
		AccountNumber: accountNumber,
		Balance:       0,
	}
	if err := u.walletRepo.Create(ctx, wallet); err != nil {
		// PERINGATAN: user sudah tersimpan pada TAHAP 3 sedangkan wallet gagal,
		// sehingga tercipta user yatim (bisa login tapi tidak punya dompet).
		// Penghapusan kompensasi tidak bisa dilakukan dari sini karena
		// domain.UserRepository tidak menyediakan Delete. Solusi yang benar
		// adalah membungkus Create(user)+Create(wallet) dalam satu transaksi
		// database (pola Unit of Work) pada lapisan repository.
		return nil, nil, fmt.Errorf("user berhasil dibuat tetapi gagal membuat wallet: %w", err)
	}

	// Sukses: kembalikan pointer user, pointer wallet, dan error nil.
	return user, wallet, nil
}

// Tujuan   : memvalidasi kredensial login pengguna.
// Penggunaan: dipanggil saat endpoint HTTP POST /api/login menerima request.
// Parameter :
//   - ctx context.Context: pembatalan/timeout dari HTTP request.
//   - email string       : email yang dimasukkan user.
//   - password string    : password mentah untuk dicocokkan.
//
// Return   :
//   - *domain.User: pointer data user jika email & password benar.
//   - error       : nil jika login sukses; error jika gagal / kredensial salah.
func (u *userUsecase) Login(ctx context.Context, email, password string) (*domain.User, error) {
	// TAHAP 0: Normalisasi email agar konsisten dengan Register.
	// Tanpa ini, pengguna yang mendaftarkan "Budi@Mail.com" tidak akan bisa
	// login dengan "budi@mail.com".
	email = strings.ToLower(strings.TrimSpace(email))

	// TAHAP 1: Ambil data user berdasarkan email.
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data user: %w", err)
	}
	if user == nil {
		// User tidak ditemukan. Demi keamanan:
		//  1. Pesan error dibuat sama dengan kasus password salah, agar
		//     penyerang tidak bisa membedakan keduanya lewat pesan.
		//  2. Tetap jalankan bcrypt dummy, agar waktu respons tidak
		//     membocorkan apakah email tersebut terdaftar
		//     (mitigasi user enumeration via timing attack).
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash(), []byte(password))
		return nil, ErrInvalidCredentials
	}

	// TAHAP 2: Bandingkan password mentah dengan password_hash yang tersimpan.
	// bcrypt.CompareHashAndPassword mengembalikan nil jika password cocok.
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		// Pesan yang sama persis dengan kasus email tidak ditemukan.
		return nil, ErrInvalidCredentials
	}

	// Sukses: kembalikan data user.
	return user, nil
}
