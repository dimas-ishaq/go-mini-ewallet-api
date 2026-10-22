package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-mini-ewallet-api/internal/usecase"
)

// WalletHandler menangani request HTTP terkait saldo dan transfer wallet.
// Handler menerjemahkan request/response HTTP dan menyerahkan aturan bisnis ke WalletUsecase.
type WalletHandler struct {
	walletUsecase usecase.WalletUsecase
}

// NewWalletHandler membuat handler wallet dengan usecase yang sudah dirangkai.
// Parameter: wUsecase menyediakan logika bisnis saldo dan transfer.
// Return: pointer handler yang dapat didaftarkan pada router Gin.
func NewWalletHandler(wUsecase usecase.WalletUsecase) *WalletHandler {
	return &WalletHandler{
		walletUsecase: wUsecase,
	}
}

// TransferRequest adalah bentuk JSON body yang diterima endpoint transfer.
// destination_account wajib diisi; amount adalah nilai uang yang diminta untuk dikirim.
type TransferRequest struct {
	DestinationAccount string  `json:"destination_account" binding:"required"`
	Amount             float64 `json:"amount" binding:"required"`
}

// GetBalance membaca saldo wallet milik user yang sudah diautentikasi.
// Tahapan: ambil userID dari konteks middleware, panggil usecase, lalu balas JSON.
// Error usecase dibalas 400; sukses dibalas 200 dengan wallet pada field data.
func (h *WalletHandler) GetBalance(c *gin.Context) {
	// Ambil userID dari payload JWT yang disimpan di context oleh AuthMiddleware.
	rawID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "pengguna belum terautentikasi"})
		return
	}
	userID, ok := rawID.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "identitas pengguna tidak valid"})
		return
	}

	wallet, err := h.walletUsecase.GetBalance(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": wallet})
}

// Transfer memproses permintaan transfer dari user yang sudah diautentikasi.
// Tahapan: ambil userID, bind dan validasi JSON, panggil usecase, lalu kirim hasil.
// Nomor rekening tujuan dan nominal berasal dari TransferRequest.
func (h *WalletHandler) Transfer(c *gin.Context) {
	rawID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "pengguna belum terautentikasi"})
		return
	}
	userID, ok := rawID.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "identitas pengguna tidak valid"})
		return
	}

	// Bind JSON body; error EOF/validasi diformat ke bahasa Indonesia.
	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	// Usecase memvalidasi aturan bisnis dan meminta repository menjalankan transfer.
	err := h.walletUsecase.Transfer(c.Request.Context(), userID, req.DestinationAccount, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Kesalahan sintaks sebelumnya berada di map gin.H: setiap pasangan key-value
	// harus dipisahkan koma, termasuk pasangan terakhir sebelum kurung penutup.
	c.JSON(http.StatusOK, gin.H{
		"message": "Transfer berhasil dilakukan",
	})
}
