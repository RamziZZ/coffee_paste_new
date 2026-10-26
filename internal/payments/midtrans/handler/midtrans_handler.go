package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"coffee_paste_backend/internal/payments/midtrans/dto"
	"coffee_paste_backend/internal/payments/midtrans/service"
	"coffee_paste_backend/pkg/midtrans"
)

type MidtransHandler struct {
	Service *service.MidtransService
}

func NewMidtransHandler(
	svc *service.MidtransService,
) *MidtransHandler {
	return &MidtransHandler{Service: svc}
}

func midtransError(c *gin.Context, err error) {
	message := err.Error()

	switch {
	case strings.Contains(message, "tidak ditemukan"):
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": message,
		})
	case strings.Contains(message, "sudah dibayar"),
		strings.Contains(message, "signature"):
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": message,
		})
	case strings.Contains(message, "harus"),
		strings.Contains(message, "wajib"),
		strings.Contains(message, "tidak valid"):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": message,
		})
	case strings.Contains(message, "Midtrans"):
		c.JSON(http.StatusBadGateway, gin.H{
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

// CreateSnap membuat Snap transaction untuk sebuah pesanan.
// POST /payments/midtrans/snap
func (h *MidtransHandler) CreateSnap(c *gin.Context) {
	var req dto.CreateSnapPaymentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Data pembayaran tidak valid",
			"error":   err.Error(),
		})
		return
	}

	result, err := h.Service.CreateSnap(c.Request.Context(), req)
	if err != nil {
		midtransError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Snap transaction berhasil dibuat",
		"data":    result,
	})
}

// HandleNotification menangani webhook notifikasi dari Midtrans.
// POST /payments/midtrans/webhook
func (h *MidtransHandler) HandleNotification(c *gin.Context) {
	var notification midtrans.Notification

	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Payload notifikasi tidak valid",
		})
		return
	}

	if notification.OrderID == "" || notification.SignatureKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Payload notifikasi tidak lengkap",
		})
		return
	}

	err := h.Service.HandleNotification(c.Request.Context(), notification)
	if err != nil {
		midtransError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.MidtransNotificationResponse{
		Success: true,
		Message: "Notifikasi berhasil diproses",
	})
}

// GetByOrderID mengambil status pembayaran Midtrans untuk pesanan.
// GET /payments/midtrans/order/:order_id
func (h *MidtransHandler) GetByOrderID(c *gin.Context) {
	orderID := c.Param("order_id")

	payment, err := h.Service.GetByOrderID(c.Request.Context(), orderID)
	if err != nil {
		midtransError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Status pembayaran berhasil diambil",
		"data":    payment,
	})
}

// GetByID mengambil detail pembayaran Midtrans.
// GET /payments/midtrans/:id
func (h *MidtransHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "ID pembayaran tidak valid",
		})
		return
	}

	payment, err := h.Service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrPaymentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Pembayaran tidak ditemukan",
			})
			return
		}
		midtransError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Detail pembayaran berhasil diambil",
		"data":    payment,
	})
}

// GetConfig mengambil konfigurasi Midtrans untuk frontend.
// GET /payments/midtrans/config
func (h *MidtransHandler) GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"client_key":  h.Service.GetClientKey(),
			"environment": h.Service.GetEnvironment(),
		},
	})
}
