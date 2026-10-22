package usecase

import (
	"context"
	"errors"
	"go-mini-ewallet-api/internal/domain"
)

// WalletUsecase menetapkan operasi bisnis untuk membaca saldo dan mentransfer dana.
// Implementasinya menggunakan WalletRepository agar terpisah dari penyimpanan.
type WalletUsecase interface {
	GetBalance(ctx context.Context, userID int) (*domain.Wallet, error)
	Transfer(ctx context.Context, senderUserID int, destAccountNumber string, amount float64) error
}

// walletUsecase menjalankan aturan bisnis wallet melalui repository.
type walletUsecase struct {
	walletRepo domain.WalletRepository
}

// NewWalletUsecase membentuk usecase wallet dengan repository yang diberikan.
// Parameter: wRepo adalah kontrak akses data wallet.
// Return: implementasi WalletUsecase.
func NewWalletUsecase(wRepo domain.WalletRepository) WalletUsecase {
	return &walletUsecase{
		walletRepo: wRepo,
	}

}

// GetBalance mengambil wallet milik user.
// Parameter: ctx untuk pembatalan/batas waktu; userID adalah ID pemilik.
// Return: wallet dan nil saat ditemukan; nil dan error bila repository gagal
// atau wallet tidak tersedia.
func (u *walletUsecase) GetBalance(ctx context.Context, userID int) (*domain.Wallet, error) {
 	// Ambil wallet sumber berdasarkan ID user dan teruskan error repository apa adanya.
	wallet, err := u.walletRepo.GetByUserID(ctx, userID)
 	// Hentikan proses jika pencarian gagal; tidak ada wallet yang dapat dikembalikan.
	if err != nil {
		return nil, err
	}

 	// Perlakukan hasil kosong sebagai wallet yang tidak ditemukan.
	if wallet == nil {
		return nil, errors.New("dompet tidak ditemukan")
	}

 	// Kembalikan wallet setelah pencarian berhasil dan hasilnya tersedia.
	return wallet, nil
}

// Transfer memvalidasi nominal, keberadaan wallet, dan tujuan sebelum transfer.
// Parameter: ctx untuk pembatalan/batas waktu; senderUserID adalah pemilik
// sumber; destAccountNumber adalah rekening tujuan; amount adalah nominal.
// Return: nil bila repository berhasil mentransfer; error jika validasi,
// pencarian wallet, atau transfer gagal.
// Tahapan: validasi nominal, ambil wallet sumber dan tujuan, tolak transfer
// ke wallet yang sama, lalu serahkan pemindahan saldo ke repository.
func (u *walletUsecase) Transfer(ctx context.Context, senderUserID int, destAccountNumber string, amount float64) error {
 	// Tolak nominal nol atau negatif sebelum melakukan akses repository.
	if amount <= 0 {
		return errors.New("nominal transfer harus lebih dari 0")
	}

 	// Cari wallet sumber milik pengirim; kegagalan pencarian menghentikan transfer.
	senderWallet, err := u.walletRepo.GetByUserID(ctx, senderUserID)
	if err != nil {
		return err
	}
 	// Pastikan pengirim memiliki wallet sebelum mencari rekening tujuan.
	if senderWallet == nil {
		return errors.New("dompet pengirim tidak ditemukan")
	}

 	// Cari wallet tujuan menggunakan nomor rekening yang diberikan.
	destWallet, err := u.walletRepo.GetByAccountNumber(ctx, destAccountNumber)
	if err != nil {
		return err
	}

 	// Tolak nomor rekening yang tidak terhubung ke wallet.
	if destWallet == nil {
		return errors.New("dompet tujuan tidak ditemukan")
	}

 	// Cegah pemindahan dana ke wallet sumber yang sama.
	if senderWallet.ID == destWallet.ID {
		return errors.New("transfer ke dompet sendiri tidak diperbolehkan")
	}

 	// Serahkan pembaruan saldo kepada repository setelah seluruh validasi lolos.
	return u.walletRepo.Transfer(ctx, senderWallet.ID, destWallet.ID, amount)
}
