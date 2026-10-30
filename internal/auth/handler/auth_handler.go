package handler

import (
	"errors"
	"net/http"

	"coffee_paste_backend/internal/auth/dto"
	"coffee_paste_backend/internal/auth/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

// LOGIN
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Email dan password wajib diisi dengan format yang benar",
		})
		return
	}
	result, err := h.service.Login(
		c.Request.Context(),
		req,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Email atau password salah",
			})
		case errors.Is(err, service.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Akun tidak aktif",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Terjadi kesalahan pada server",
			})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Login berhasil",
		"data":    result,
	})
}

// ME
func (h *AuthHandler) Me(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "User ID tidak ditemukan dalam token",
		})
		return
	}
	userID, ok := userIDValue.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "User ID dalam token tidak valid",
		})
		return
	}
	result, err := h.service.Me(
		c.Request.Context(),
		userID,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Akun tidak aktif",
			})
		case errors.Is(err, service.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "User tidak valid",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Terjadi kesalahan pada server",
			})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Data user berhasil diambil",
		"data":    result,
	})
}

// FORGOT PASSWORD
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Email wajib diisi dengan format yang benar",
		})
		return
	}
	token, err := h.service.ForgotPassword(
		c.Request.Context(),
		req,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Akun tidak aktif",
			})
		case errors.Is(err, service.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Email atau password salah",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Terjadi kesalahan pada server",
			})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Token reset password berhasil dibuat",
		"data": gin.H{
			"reset_token": token,
		},
	})
}

// RESET PASSWORD
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Token dan password baru wajib diisi",
		})
		return
	}
	err := h.service.ResetPassword(
		c.Request.Context(),
		req,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPasswordResetToken):
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Token reset password tidak valid atau sudah kedaluwarsa",
			})
		case errors.Is(err, service.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Akun tidak aktif",
			})
		case errors.Is(err, service.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Email atau password salah",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Gagal mengubah password",
			})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password berhasil diubah",
	})
}

func (h *AuthHandler) RegisterCustomer(c *gin.Context) {
	var req dto.RegisterCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Data register tidak valid",
		})
		return
	}
	result, err := h.service.RegisterCustomer(
		c.Request.Context(),
		req,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"message": "Email sudah terdaftar",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Gagal membuat akun customer",
			})
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Registrasi customer berhasil",
		"data":    result,
	})
}

// GOOGLE LOGIN
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	var req dto.GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "ID token Google wajib diisi",
		})
		return
	}
	result, err := h.service.LoginWithGoogle(
		c.Request.Context(),
		req,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGoogleNotConfigured):
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"message": "Login Google belum dikonfigurasi di server",
			})
		case errors.Is(err, service.ErrInvalidGoogleToken):
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Token Google tidak valid atau sudah kedaluwarsa",
			})
		case errors.Is(err, service.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Akun tidak aktif",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Terjadi kesalahan pada server",
			})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Login dengan Google berhasil",
		"data":    result,
	})
}
