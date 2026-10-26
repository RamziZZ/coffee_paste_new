package handler

import (
	"errors"
	"net/http"
	"strings"

	"coffee_paste_backend/internal/menu/dto"
	"coffee_paste_backend/internal/menu/repository"
	"coffee_paste_backend/internal/menu/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MenuHandler struct {
	service *service.MenuService
}

func NewMenuHandler(svc *service.MenuService) *MenuHandler {
	return &MenuHandler{service: svc}
}

func menuError(c *gin.Context, err error) {
	message := err.Error()

	switch {
	case errors.Is(err, repository.ErrInUse):
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": message,
		})
	case errors.Is(err, repository.ErrNotFound),
		strings.Contains(message, "tidak ditemukan"):
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": message,
		})
	case strings.Contains(message, "tidak valid"),
		strings.Contains(message, "wajib"),
		strings.Contains(message, "harus"),
		strings.Contains(message, "tidak boleh"):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": message,
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Terjadi kesalahan pada server",
		})
	}
}

// GET /api/v1/menu
func (h *MenuHandler) GetAllMenu(c *gin.Context) {
	menus, err := h.service.GetAll(
		c.Request.Context(),
		c.Query("q"),
		c.Query("category_id"),
		c.Query("status"),
	)
	if err != nil {
		menuError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Daftar menu berhasil diambil",
		"data":    menus,
	})
}

// GET /api/v1/menu/:id
func (h *MenuHandler) GetMenuByID(c *gin.Context) {
	menu, err := h.service.GetByID(
		c.Request.Context(),
		c.Param("id"),
	)
	if err != nil {
		menuError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Detail menu berhasil diambil",
		"data":    menu,
	})
}

// POST /api/v1/menu
func (h *MenuHandler) CreateMenu(c *gin.Context) {
	var req dto.CreateMenuRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Data menu tidak valid",
			"error":   err.Error(),
		})
		return
	}

	menu, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		menuError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Menu berhasil ditambahkan",
		"data":    menu,
	})
}

// PUT /api/v1/menu/:id
func (h *MenuHandler) UpdateMenu(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "ID menu tidak valid",
		})
		return
	}

	var req dto.UpdateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Data menu tidak valid",
			"error":   err.Error(),
		})
		return
	}

	menu, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		menuError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Menu berhasil diperbarui",
		"data":    menu,
	})
}

// DELETE /api/v1/menu/:id
func (h *MenuHandler) DeleteMenu(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "ID menu tidak valid",
		})
		return
	}

	err := h.service.Delete(c.Request.Context(), id)
	if err != nil {
		menuError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Menu berhasil dihapus",
	})
}

func (h *MenuHandler) GetFeaturedMenu(c *gin.Context) {
	menus, err := h.service.GetFeaturedMenu(c.Request.Context())
	if err != nil {
		menuError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Menu featured berhasil diambil",
		"data":    menus,
	})
}

func (h *MenuHandler) GetCategories(c *gin.Context) {
	categories, err := h.service.GetCategories(c.Request.Context())
	if err != nil {
		menuError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Kategori menu berhasil diambil",
		"data":    categories,
	})
}

func (h *MenuHandler) GetFilters(c *gin.Context) {
	filters, err := h.service.GetFilters(c.Request.Context())
	if err != nil {
		menuError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Filter menu berhasil diambil",
		"data":    filters,
	})
}
