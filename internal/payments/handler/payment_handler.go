package handler

import (
	"errors"
	"net/http"
	"strings"

	"coffee_paste_backend/internal/payments/dto"
	"coffee_paste_backend/internal/payments/repository"
	"coffee_paste_backend/internal/payments/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PaymentHandler struct {
	Service *service.PaymentService
}

func NewPaymentHandler(
	svc *service.PaymentService,
) *PaymentHandler {
	return &PaymentHandler{Service: svc}
}

func paymentError(c *gin.Context, err error) {
	message := err.Error()

	switch {
	case errors.Is(err, repository.ErrOrderNotFound),
		errors.Is(err, repository.ErrPaymentNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": message,
		})
	case errors.Is(err, repository.ErrOrderAlreadyPaid):
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": message,
		})
	case errors.Is(err, repository.ErrAmountNotMatch),
		errors.Is(err, repository.ErrCashBelowAmount),
		errors.Is(err, repository.ErrInvalidTransition),
		errors.Is(err, repository.ErrCashNotApplicable),
		errors.Is(err, repository.ErrNoChangeForNonCash),
		strings.Contains(message, "tidak ditemukan"),
		strings.Contains(message, "harus"),
		strings.Contains(message, "tidak valid"),
		strings.Contains(message, "tidak boleh"),
		strings.Contains(message, "sudah"),
		strings.Contains(message, "hanya dapat"),
		strings.Contains(message, "hanya untuk"),
		strings.Contains(message, "tidak diizinkan"):
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

// getActor mengambil user_id (UUID string) yang diset oleh AuthMiddleware.
func getActor(c *gin.Context) (string, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "User ID tidak ditemukan",
		})
		return "", false
	}

	userID, ok := value.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "User ID tidak valid",
		})
		return "", false
	}

	return userID, true
}

func (h *PaymentHandler) Create(c *gin.Context) {
	receivedBy, ok := getActor(c)
	if !ok {
		return
	}

	var req dto.CreatePaymentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Data pembayaran tidak valid",
			"error":   err.Error(),
		})
		return
	}

	payment, err := h.Service.Create(c.Request.Context(), req, receivedBy)
	if err != nil {
		paymentError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Pembayaran berhasil dibuat",
		"data":    payment,
	})
}

func (h *PaymentHandler) GetByID(c *gin.Context) {
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
		paymentError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Detail pembayaran berhasil diambil",
		"data":    payment,
	})
}

func (h *PaymentHandler) GetByOrderID(c *gin.Context) {
	orderID := c.Param("id")
	if _, err := uuid.Parse(orderID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "ID pesanan tidak valid",
		})
		return
	}

	payments, err := h.Service.GetByOrderID(
		c.Request.Context(),
		orderID,
	)
	if err != nil {
		paymentError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Daftar pembayaran berhasil diambil",
		"data":    payments,
	})
}

func (h *PaymentHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "ID pembayaran tidak valid",
		})
		return
	}

	var req dto.UpdatePaymentStatusRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Data status pembayaran tidak valid",
			"error":   err.Error(),
		})
		return
	}

	payment, err := h.Service.UpdateStatus(
		c.Request.Context(),
		id,
		req,
	)
	if err != nil {
		paymentError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Status pembayaran berhasil diperbarui",
		"data":    payment,
	})
}
