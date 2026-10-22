package domain

// import: ambil paket lain yang dipakai file ini.
// "context" -> tipe context.Context untuk timeout/cancel DB.
// "time"    -> tipe time.Time untuk field CreatedAt.
import (
	"context"
	"time"
)

// Wallet dompet milik satu user. Satu user = satu wallet (di proyek ini).
type Wallet struct {
	// ID nomor unik wallet (primary key).
	ID int `json:"id"`
	// UserID pemilik wallet. Relasi ke User.ID.
	UserID int `json:"user_id"`
	// AccountNumber nomor unik wallet yang ditampilkan ke user.
	AccountNumber string `json:"account_number"`
	// Balance saldo saat ini. float64 karena ada desimal (Rupiah sen).
	Balance float64 `json:"balance"`
	// CreatedAt waktu wallet dibuat.
	CreatedAt time.Time `json:"created_at"`
}

// Transaction catat setiap perpindahan uang.
// Dipakai untuk topup, transfer, withdraw.
type Transaction struct {
	// ID unik transaksi. int64 karena jumlah transaksi bisa sangat banyak.
	ID int64 `json:"id"`
	// SourceWalletID wallet pengirim. Pakai *int (pointer) agar bisa nil.
	// nil = tidak ada pengirim (contoh: topup dari luar sistem).
	// omitempty = kalau nil, field tidak muncul di JSON.
	SourceWalletID *int `json:"source_wallet_id,omitempty"`
	// DestinationWalletID wallet penerima. *int juga agar bisa nil.
	// nil = tidak ada penerima (contoh: withdraw keluar sistem).
	DestinationWalletID *int `json:"destination_wallet_id,omitempty"`
	// Amount nominal transaksi.
	Amount float64 `json:"amount"`
	// TransactionType jenis: "topup", "transfer", "withdraw".
	TransactionType string `json:"transaction_type"`
	// CreatedAt waktu transaksi dicatat.
	CreatedAt time.Time `json:"created_at"`
}

// WalletRepository kontrak akses data wallet & transaksi.
// Sama seperti UserRepository: hanya daftar fungsi, isi ada di postgres.
type WalletRepository interface {
	// Tujuan   : buat wallet baru untuk user.
	// Penggunaan: panggil saat user baru dibuat.
	// Parameter : ctx untuk cancel/timeout; wallet berisi pemilik dan saldo awal.
	// Return   : nil jika berhasil; error jika insert gagal.
	Create(ctx context.Context, wallet *Wallet) error
	// Tujuan   : cari wallet berdasarkan pemiliknya.
	// Penggunaan: panggil saat perlu membaca wallet user.
	// Parameter : ctx untuk cancel/timeout; userID adalah ID pemilik.
	// Return   : pointer wallet jika ditemukan; nil,nil jika tidak ada;
	//            nil,error jika database bermasalah.
	GetByUserID(ctx context.Context, userID int) (*Wallet, error)
	// Tujuan   : cari wallet berdasarkan nomor rekening.
	// Penggunaan: panggil saat transfer menerima nomor rekening tujuan.
	// Parameter : ctx untuk cancel/timeout; accNum adalah nomor yang dicari.
	// Return   : pointer wallet jika ditemukan; nil,nil jika tidak ada;
	//            nil,error jika database bermasalah.
	GetByAccountNumber(ctx context.Context, accNum string) (*Wallet, error)
	// Tujuan   : pindahkan sejumlah saldo dari wallet sumber ke tujuan.
	// Penggunaan: panggil setelah memvalidasi pemilik dan nominal transfer.
	// Parameter : ctx untuk cancel/timeout; sourceWalletID dan destWalletID
	//            adalah ID wallet; amount adalah nominal yang dipindahkan.
	// Return   : nil jika berhasil; error jika transfer gagal. Implementasi
	//            sebaiknya memakai transaksi DB agar saldo konsisten.
	Transfer(ctx context.Context, sourceWalletID, destWalletID int, amount float64) error
}
