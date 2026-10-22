package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// formatBindError mengubah error decoding dan binding validator Gin menjadi
// pesan bahasa Indonesia yang mudah dipahami pengguna.
func formatBindError(err error) string {
	if err == nil {
		return ""
	}

	// Kasus 1: Body kosong (EOF)
	if errors.Is(err, io.EOF) {
		return "data permintaan tidak boleh kosong"
	}

	// Kasus 2: Format JSON salah ketik (syntax/unmarshal)
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return "format JSON tidak valid"
	}
	var unmarshalTypeErr *json.UnmarshalTypeError
	if errors.As(err, &unmarshalTypeErr) {
		return fmt.Sprintf("tipe data untuk field '%s' tidak sesuai", unmarshalTypeErr.Field)
	}

	// Kasus 3: Validasi tag (required, email, min, max, dsb)
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, fe := range ve {
			field := fe.Field()
			switch fe.Tag() {
			case "required":
				return fmt.Sprintf("field '%s' wajib diisi", field)
			case "email":
				return "format email tidak valid"
			case "min":
				return fmt.Sprintf("field '%s' minimal %s karakter/nilai", field, fe.Param())
			case "max":
				return fmt.Sprintf("field '%s' maksimal %s karakter/nilai", field, fe.Param())
			case "gt":
				return fmt.Sprintf("field '%s' harus lebih besar dari %s", field, fe.Param())
			default:
				return fmt.Sprintf("field '%s' tidak memenuhi syarat (%s)", field, fe.Tag())
			}
		}
	}

	return "permintaan data tidak valid"
}

// respondBadRequest merespons dengan HTTP 400 beserta pesan kesalahan berbahasa Indonesia.
func respondBadRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"error": formatBindError(err)})
}
