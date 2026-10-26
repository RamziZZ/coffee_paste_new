package handler

import (
	"errors"
	"net/http"

	"coffee_paste_backend/internal/order_items/dto"
	"coffee_paste_backend/internal/order_items/repository"
	"coffee_paste_backend/internal/order_items/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrderItemHandler struct {
	service *service.OrderItemService
}

func NewOrderItemHandler(s *service.OrderItemService) *OrderItemHandler {
	return &OrderItemHandler{service: s}
}

func parseUUID(c *gin.Context, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": param + " harus berupa UUID yang valid",
		})
		return uuid.Nil, false
	}

	return id, true
}

func (h *OrderItemHandler) GetByOrderID(c *gin.Context) {
	orderID, ok := parseUUID(c, "order_id")
	if !ok {
		return
	}

	items, err := h.service.GetByOrderID(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Gagal mengambil item pesanan",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    items,
	})
}

func (h *OrderItemHandler) Create(c *gin.Context) {
	orderID, ok := parseUUID(c, "order_id")
	if !ok {
		return
	}

	var req dto.CreateOrderItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Request tidak valid: " + err.Error(),
		})
		return
	}

	item, err := h.service.Create(c.Request.Context(), orderID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Item pesanan berhasil ditambahkan",
		"data":    item,
	})
}

func (h *OrderItemHandler) Update(c *gin.Context) {
	orderID, ok := parseUUID(c, "order_id")
	if !ok {
		return
	}

	itemID, ok := parseUUID(c, "item_id")
	if !ok {
		return
	}

	var req dto.UpdateOrderItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Request tidak valid: " + err.Error(),
		})
		return
	}

	item, err := h.service.Update(
		c.Request.Context(),
		orderID,
		itemID,
		req,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Item pesanan berhasil diperbarui",
		"data":    item,
	})
}

func (h *OrderItemHandler) Delete(c *gin.Context) {
	orderID, ok := parseUUID(c, "order_id")
	if !ok {
		return
	}

	itemID, ok := parseUUID(c, "item_id")
	if !ok {
		return
	}

	err := h.service.Delete(c.Request.Context(), orderID, itemID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Item pesanan berhasil dihapus",
	})
}

func (h *OrderItemHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
	case errors.Is(err, repository.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": err.Error(),
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Terjadi kesalahan pada server",
		})
	}
}
