package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go-mini-ewallet-api/internal/domain"
)

// postgresWalletRepository mengimplementasikan domain.WalletRepository untuk PostgreSQL.
// db adalah pool koneksi bersama; operasi query menerima context dari pemanggil.
type postgresWalletRepository struct {
	db *sql.DB
}

// Tujuan   : buat wallet repository Postgres yang memenuhi domain.WalletRepository.
// Penggunaan: panggil saat merangkai aplikasi, lalu berikan hasilnya ke service.
// Parameter : db adalah pool koneksi database yang sudah dibuka.
// Return   : domain.WalletRepository yang menyediakan operasi data wallet.
func NewPostgresWalletRepository(db *sql.DB) domain.WalletRepository {
	return &postgresWalletRepository{db: db}
}

// Tujuan   : simpan wallet baru ke database.
// Penggunaan: panggil saat membuat wallet untuk user.
// Parameter : ctx untuk timeout/cancel query; wallet berisi data wallet.
// Return   : nil jika berhasil; error jika penyimpanan gagal.
// Alur: insert data wallet, lalu ambil ID dan waktu pembuatan dari RETURNING.
func (r *postgresWalletRepository) Create(ctx context.Context, wallet *domain.Wallet) error {
	// Query untuk insert baris baru wallet dan mengembalikan id serta created_at.
	query := `INSERT INTO wallets (user_id, account_number, balance) VALUES ($1, $2, $3) RETURNING id, created_at`

	// QueryRowContext hanya mengirim query; Scan harus dipanggil untuk mengeksekusi
	// dan mengambil hasil RETURNING. fmt.Errorf dengan %w dipakai untuk membungkus error.
	err := r.db.QueryRowContext(ctx, query, wallet.UserID, wallet.AccountNumber, wallet.Balance).Scan(&wallet.ID, &wallet.CreatedAt)
	if err != nil {
		return fmt.Errorf("gagal insert wallet: %w", err)
	}
	return nil
}

// Tujuan   : ambil wallet berdasarkan pemilik.
// Penggunaan: panggil saat membaca wallet dari user.
// Parameter : ctx untuk timeout/cancel query; userID adalah ID pemilik.
// Return   : pointer wallet jika ditemukan; nil,nil jika tidak ada; nil,error jika gagal.
// Alur: query satu baris berdasarkan user_id; ErrNoRows berarti wallet belum ada.
func (r *postgresWalletRepository) GetByUserID(ctx context.Context, userID int) (*domain.Wallet, error) {
	// Query baris wallet berdasarkan user_id.
	query := `SELECT id,user_id,account_number,balance,created_at FROM wallets WHERE user_id = $1`

	var wallet domain.Wallet
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&wallet.ID, &wallet.UserID, &wallet.AccountNumber, &wallet.Balance, &wallet.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal get wallet by user id :%w", err)
	}
	return &wallet, nil
}

// Tujuan   : ambil wallet berdasarkan nomor rekening.
// Penggunaan: panggil saat lookup wallet dari nomor rekening transfer.
// Parameter : ctx untuk timeout/cancel query; accNum adalah nomor rekening.
// Return   : pointer wallet jika ditemukan; nil,nil jika tidak ada; nil,error jika gagal.
// Alur: query satu baris berdasarkan account_number; ErrNoRows berarti tidak ditemukan.
func (r *postgresWalletRepository) GetByAccountNumber(ctx context.Context, accNum string) (*domain.Wallet, error) {
	// Query baris wallet berdasarkan account_number unik.
	query := `SELECT id,user_id, account_number,balance,created_at FROM wallets WHERE account_number = $1`

	var wallet domain.Wallet

	err := r.db.QueryRowContext(ctx, query, accNum).Scan(&wallet.ID, &wallet.UserID, &wallet.AccountNumber, &wallet.Balance, &wallet.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal get wallet by account number :%w", err)
	}
	return &wallet, nil
}

// Tujuan   : pindahkan saldo antar wallet.
// Penggunaan: panggil saat memproses transfer; pakai transaksi DB agar atomik.
// Parameter : ctx untuk timeout/cancel; sourceWalletID dan destWalletID adalah wallet
//
//	sumber dan tujuan; amount adalah nominal.
//
// Return   : nil jika berhasil; error jika transfer gagal.
// Tahapan: mulai transaksi, kunci saldo sumber dan tujuan, validasi saldo,
// ubah saldo kedua wallet, catat transaksi, lalu commit. Rollback otomatis
// dijalankan saat fungsi keluar sebelum commit berhasil.
func (r *postgresWalletRepository) Transfer(ctx context.Context, sourceWalletID, destWalletID int, amount float64) error {
	// Mulai transaksi agar perubahan saldo dan pencatatan riwayat bersifat atomik.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("gagal create transaction: %w", err)
	}
	// Rollback otomatis saat fungsi keluar sebelum Commit berhasil.
	defer tx.Rollback()

	// Kunci baris pengirim agar saldo tidak berubah oleh transfer bersamaan.
	var sourceBalance float64
	err = tx.QueryRowContext(ctx, "SELECT balance FROM wallets WHERE id = $1 FOR UPDATE", sourceWalletID).Scan(&sourceBalance)
	if err != nil {
		return fmt.Errorf("gagal lock source wallet: %w", err)
	}

	// Tolak transfer bila saldo pengirim kurang dari nominal.
	if sourceBalance < amount {
		return errors.New("saldo tidak mencukupi")
	}

	// Kunci baris penerima sebelum perubahan saldo.
	var destBalance float64
	err = tx.QueryRowContext(ctx, "SELECT balance FROM wallets WHERE id = $1 FOR UPDATE", destWalletID).Scan(&destBalance)
	if err != nil {
		return fmt.Errorf("gagal lock destination wallet: %w", err)
	}

	// Kurangi saldo pengirim dan tambahkan saldo penerima.
	_, err = tx.ExecContext(ctx, "UPDATE wallets SET balance = balance - $1 WHERE id = $2", amount, sourceWalletID)
	if err != nil {
		return fmt.Errorf("gagal potong saldo pengirim: %w", err)
	}
	_, err = tx.ExecContext(ctx, "UPDATE wallets SET balance = balance + $1 WHERE id = $2", amount, destWalletID)
	if err != nil {
		return fmt.Errorf("gagal tambah saldo penerima: %w", err)
	}

	// Simpan riwayat transfer dalam transaksi yang sama.
	_, err = tx.ExecContext(ctx, `INSERT INTO transactions (source_wallet_id, destination_wallet_id, amount, transaction_type) VALUES ($1,$2,$3, 'TRANSFER')`, sourceWalletID, destWalletID, amount)
	if err != nil {
		return fmt.Errorf("gagal catat riwayat transaksi:%w", err)
	}

	// Commit membuat perubahan saldo dan riwayat berlaku bersama-sama.
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit transaksi: %w", err)
	}
	return nil
}
