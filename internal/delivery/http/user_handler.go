package http

// import: daftar pustaka yang digunakan oleh user handler.
// "net/http"                          -> konstanta status kode HTTP (200 OK, 201 Created, 400 Bad Request, dsb).
// "github.com/gin-gonic/gin"          -> framework web HTTP Gin (Context, JSON response, ShouldBindJSON).
// "go-mini-ewallet-api/internal/usecase" -> interface UserUsecase untuk pemanggilan logika bisnis user.
import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-mini-ewallet-api/internal/usecase"
)

// UserHandler bertindak sebagai controller lapisan HTTP untuk fitur pengguna (User).
// Handler bertugas menerima HTTP request, validasi format JSON (binding), memanggil Usecase,
// dan mengembalikan HTTP response yang sesuai kepada client.
type UserHandler struct {
	userUsecase usecase.UserUsecase
}

// Tujuan   : membuat instance baru dari UserHandler (constructor).
// Penggunaan: dipanggil saat setup route HTTP di cmd/api/main.go.
// Parameter : userUsecase adalah implementasi usecase yang sudah diinisialisasi.
// Return   : pointer *UserHandler yang siap didaftarkan ke router Gin.
func NewUserHandler(userUsecase usecase.UserUsecase) *UserHandler {
	return &UserHandler{
		userUsecase: userUsecase,
	}
}

// RegisterRequest mendefinisikan skema JSON body untuk endpoint registrasi.
// Tag `binding:"required,..."` dipakai oleh Gin validator untuk memastikan field tidak kosong
// serta mematuhi kriteria format email dan batas panjang karakter sebelum masuk ke usecase.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=100"`
}

// LoginRequest mendefinisikan skema JSON body untuk endpoint login pengguna.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=100"`
}

// Tujuan   : menangani HTTP POST /api/register untuk membuat akun baru.
// Penggunaan: didaftarkan ke Gin router: r.POST("/api/register", handler.Register).
// Parameter : c *gin.Context yang menyimpan request payload, context cancel/timeout, dan response writer.
// Return   : tidak mengembalikan return value Go; membalas HTTP response JSON via c.JSON().
//
// Tahapan:
// 1. Bind JSON request body ke struct RegisterRequest. Jika data tidak sesuai tag binding, balas 400 Bad Request.
// 2. Teruskan context dan data registrasi ke userUsecase.Register.
// 3. Jika usecase mengembalikan error (misal email sudah terdaftar), balas 400 Bad Request.
// 4. Jika sukses, balas HTTP 201 Created dengan JSON berisi info user dan wallet baru.
func (h *UserHandler) Register(c *gin.Context) {
	// 1. Parsing & validasi format input JSON dari request body
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	// 2. Eksekusi logika bisnis registrasi akun dan pembuatan wallet awal
	user, wallet, err := h.userUsecase.Register(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 3. Kirim respon HTTP 201 Created
	c.JSON(http.StatusCreated, gin.H{
		"message": "Registrasi Berhasil",
		"user":    user,
		"wallet":  wallet,
	})
}

// Tujuan   : menangani HTTP POST /api/login untuk otentikasi user dan menerbitkan JWT token.
// Penggunaan: didaftarkan ke Gin router: r.POST("/api/login", handler.Login).
// Parameter : c *gin.Context pembungkus HTTP request dan response.
// Return   : respon HTTP JSON.
//
// Tahapan:
// 1. Bind dan validasi payload email dan password dari request body.
// 2. Panggil userUsecase.Login untuk memverifikasi kecocokan email dan hash password.
// 3. Bila kredensial tidak valid, balas HTTP 401 Unauthorized.
// 4. Jika login lolos, terbitkan token JWT melalui GenerateToken(userID).
// 5. Balas HTTP 200 OK beserta token JWT untuk otorisasi endpoint berikutnya.
func (h *UserHandler) Login(c *gin.Context) {
	// 1. Parsing payload login
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	// 2. Validasi kredensial pengguna via usecase
	user, err := h.userUsecase.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 3. Buat JWT token untuk session otentikasi user
	token, err := GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membuat token akses, silakan coba lagi"})
		return
	}

	// 4. Kirim respon sukses HTTP 200 OK
	c.JSON(http.StatusOK, gin.H{
		"message": "Login Berhasil",
		"user":    user,
		"token":   token,
	})
}
