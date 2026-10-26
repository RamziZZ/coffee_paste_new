package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"coffee_paste_backend/internal/favorites/service"
)

type FavoriteHandler struct {
	service *service.FavoriteService
}

func NewFavoriteHandler(service *service.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{
		service: service,
	}
}

// getUserID mengambil user_id (UUID string) yang diset oleh AuthMiddleware.
func getUserID(c *gin.Context) (string, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "User ID tidak ditemukan",
		})
		return "", false
	}

	userID, ok := value.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "User ID tidak valid",
		})
		return "", false
	}

	return userID, true
}

// writeError menerjemahkan error service ke response HTTP.
func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotCustomer):
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Akun customer tidak ditemukan",
		})
	case errors.Is(err, service.ErrMenuNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Menu tidak ditemukan",
		})
	case errors.Is(err, service.ErrAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{
			"status":  "error",
			"message": "Menu sudah ada di favorite",
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Terjadi kesalahan pada server",
		})
	}
}

// GetFavorites mengambil semua favorite milik customer yang sedang login.
func (h *FavoriteHandler) GetFavorites(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	favorites, err := h.service.GetByUserID(
		c.Request.Context(),
		userID,
	)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   favorites,
	})
}

// CreateFavorite menambahkan menu ke favorite customer.
func (h *FavoriteHandler) CreateFavorite(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	menuID := c.Param("menu_id")
	if menuID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Menu ID wajib diisi",
		})
		return
	}

	favorite, err := h.service.Create(
		c.Request.Context(),
		userID,
		menuID,
	)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data":   favorite,
	})
}

// DeleteFavorite menghapus menu dari favorite customer.
func (h *FavoriteHandler) DeleteFavorite(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	menuID := c.Param("menu_id")
	if menuID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Menu ID wajib diisi",
		})
		return
	}

	err := h.service.Delete(
		c.Request.Context(),
		userID,
		menuID,
	)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Favorite berhasil dihapus",
	})
}
