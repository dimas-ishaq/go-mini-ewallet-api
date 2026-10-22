package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// jwtSecret menjadi kunci penandatanganan token JWT.
var jwtSecret = []byte("secret")

// GenerateToken membuat JWT bertanda tangan HS256 untuk user.
// Parameter: userID adalah ID pengguna yang dimasukkan ke claim token.
// Return: token bertanda tangan dan nil jika berhasil; string kosong dan
// error bila proses penandatanganan gagal.
// Claims berisi payload standar yang ditanamkan ke dalam token JWT.
// Terdiri dari ID pengguna dan batas waktu kadaluarsa (24 jam dari pembuatan).
func GenerateToken(userID int) (string, error) {
	// Siapkan klaim token: user_id untuk identifikasi subjek, exp untuk masa berlaku.
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	// Gunakan algoritma signing HMAC-SHA256 bersama klaim payload yang telah disusun.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Tandatangani token dengan jwtSecret untuk menghasilkan representasi string siap kirim.
	return token.SignedString(jwtSecret)
}

// AuthMiddleware menyediakan middleware untuk memeriksa header Authorization.
// Penggunaan: pasang pada route yang memerlukan autentikasi.
// Return: handler Gin yang menghentikan request dan membalas 401 bila
// header Authorization kosong.
// Tahapan: periksa header; jika kosong, kirim respons unauthorized,
// batalkan rantai handler, lalu hentikan eksekusi request.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ambil nilai header HTTP Authorization dari request yang masuk.
		authHeader := c.GetHeader("Authorization")
		// Jika header tidak disediakan, tolak akses dan hentikan proses.
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "header otorisasi wajib diisi dengan format Bearer <token>"})
			c.Abort()
			return
		}

		// Buang awalan skema "Bearer " untuk mengekstrak string token mentah.
		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
		// Parse token serta verifikasi integritas tanda tangan menggunakan secret key.
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		// Pastikan parsing berhasil dan token belum melewati masa berlaku (expired).
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token tidak valid atau sudah kedaluwarsa"})
			c.Abort()
			return
		}

		// Ekstrak klaim bertipe MapClaims dari token yang terverifikasi.
		claims, ok := token.Claims.(jwt.MapClaims)
		// Tolak request jika struktur klaim tidak sesuai format MapClaims.
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "klaim token tidak valid"})
			c.Abort()
			return
		}
		// Konversi klaim user_id dari tipe float64 (standar JSON decode) ke integer.
		userID := int(claims["user_id"].(float64))
		// Simpan userID ke dalam context Gin agar bisa diakses oleh handler selanjutnya.
		c.Set("userID", userID)
		// Teruskan penanganan request ke middleware atau handler berikutnya dalam rantai pemrosesan.
		c.Next()

	}
}
