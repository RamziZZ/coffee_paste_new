package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"coffee_paste_backend/internal/inventory/dto"
	"coffee_paste_backend/internal/inventory/model"
	"coffee_paste_backend/internal/inventory/service"
)

type InventoryHandler struct {
	service *service.InventoryService
}

func NewInventoryHandler(service *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{
		service: service,
	}
}

// menuURI memvalidasi parameter :menu_id di URL harus berupa UUID.
type menuURI struct {
	MenuID string `uri:"menu_id" binding:"required,uuid"`
}

func respondBadRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{
		"status":  "error",
		"message": "Data request tidak valid",
		"detail":  err.Error(),
	})
}

// respondError memetakan error service ke status HTTP yang sesuai.
// Detail error internal hanya dicatat di log, tidak dikirim ke client.
func respondError(c *gin.Context, err error, fallbackMessage string) {
	var validationErr *model.ValidationError

	switch {
	case errors.As(err, &validationErr):
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": validationErr.Message,
		})
	case errors.Is(err, model.ErrStockNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Stok menu tidak ditemukan",
		})
	case errors.Is(err, model.ErrMenuNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Menu tidak ditemukan",
		})
	case errors.Is(err, model.ErrInsufficientStock):
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Stok tidak mencukupi untuk penyesuaian ini",
		})
	default:
		log.Println(fallbackMessage+":", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": fallbackMessage,
		})
	}
}

// List mengambil daftar stok menu.
// GET /api/v1/inventory?low_stock=&page=&limit=
func (h *InventoryHandler) List(c *gin.Context) {
	var query dto.ListInventoryQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		respondBadRequest(c, err)
		return
	}

	stocks, meta, err := h.service.List(c.Request.Context(), query)
	if err != nil {
		respondError(c, err, "Gagal mengambil daftar stok")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stocks,
		"meta":   meta,
	})
}

// Get mengambil stok satu menu.
// GET /api/v1/inventory/:menu_id
func (h *InventoryHandler) Get(c *gin.Context) {
	var uri menuURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Menu ID tidak valid",
		})
		return
	}

	stock, err := h.service.Get(c.Request.Context(), uri.MenuID)
	if err != nil {
		respondError(c, err, "Gagal mengambil stok menu")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stock,
	})
}

// Adjust menyesuaikan stok menu (set atau inkremental).
// POST /api/v1/inventory/adjust
func (h *InventoryHandler) Adjust(c *gin.Context) {
	var req dto.AdjustStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	stock, err := h.service.Adjust(c.Request.Context(), req)
	if err != nil {
		respondError(c, err, "Gagal menyesuaikan stok")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stock,
	})
}
