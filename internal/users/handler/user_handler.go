package handler

import (
	"errors"
	"net/http"

	"coffee_paste_backend/internal/users/dto"
	"coffee_paste_backend/internal/users/repository"
	"coffee_paste_backend/internal/users/service"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{service: svc}
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "User tidak ditemukan",
		})
	case errors.Is(err, service.ErrInvalidUserData):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Data user tidak valid",
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Terjadi kesalahan pada server",
		})
	}
}

func currentUserID(c *gin.Context) (string, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		return "", false
	}

	id, ok := value.(string)
	if !ok || id == "" {
		return "", false
	}

	return id, true
}

// GET /api/v1/users/me
func (h *UserHandler) Me(c *gin.Context) {
	id, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "User ID dalam token tidak valid",
		})
		return
	}

	user, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Akun tidak aktif",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Profil berhasil diambil",
		"data":    user,
	})
}

// PUT /api/v1/users/me
func (h *UserHandler) UpdateMe(c *gin.Context) {
	id, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "User ID dalam token tidak valid",
		})
		return
	}

	var req dto.UpdateMyProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Nama wajib diisi dan format data harus benar",
		})
		return
	}

	user, err := h.service.UpdateMyProfile(
		c.Request.Context(),
		id,
		&req,
	)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Profil berhasil diperbarui",
		"data":    user,
	})
}

// GET /api/v1/users
func (h *UserHandler) List(c *gin.Context) {
	users, err := h.service.List(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Daftar user berhasil diambil",
		"data":    users,
	})
}

// GET /api/v1/users/:id
func (h *UserHandler) GetByID(c *gin.Context) {
	user, err := h.service.GetByID(
		c.Request.Context(),
		c.Param("id"),
	)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Data user berhasil diambil",
		"data":    user,
	})
}

// PATCH /api/v1/users/:id/status
func (h *UserHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")

	currentID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "User ID dalam token tidak valid",
		})
		return
	}

	if currentID == id {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Admin tidak dapat mengubah status akunnya sendiri",
		})
		return
	}

	var req dto.UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "is_active wajib diisi dengan boolean",
		})
		return
	}

	user, err := h.service.UpdateStatus(
		c.Request.Context(),
		id,
		req.IsActive,
	)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Status user berhasil diperbarui",
		"data":    user,
	})
}
