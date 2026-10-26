package handler

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"coffee_paste_backend/internal/orders/dto"
	"coffee_paste_backend/internal/orders/model"
	"coffee_paste_backend/internal/orders/service"
)

type OrderHandler struct {
	service *service.OrderService
}

func NewOrderHandler(service *service.OrderService) *OrderHandler {
	return &OrderHandler{
		service: service,
	}
}

// orderURI memvalidasi parameter :id di URL harus berupa UUID.
type orderURI struct {
	ID string `uri:"id" binding:"required,uuid"`
}

// getActor mengambil user_id dan role yang diset oleh AuthMiddleware.
// Kalau gagal, response 401 langsung dikirim dan ok = false.
func getActor(c *gin.Context) (service.Actor, bool) {
	userValue, userExists := c.Get("user_id")
	roleValue, roleExists := c.Get("role")

	if !userExists || !roleExists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Data user tidak ditemukan",
		})
		return service.Actor{}, false
	}

	var userID string

	switch v := userValue.(type) {
	case string:
		userID = v
	case fmt.Stringer:
		userID = v.String()
	}

	role, _ := roleValue.(string)

	if userID == "" || role == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Data user tidak valid",
		})
		return service.Actor{}, false
	}

	return service.Actor{UserID: userID, Role: role}, true
}

// getOrderID membaca dan memvalidasi parameter :id.
func getOrderID(c *gin.Context) (string, bool) {
	var uri orderURI

	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Order ID tidak valid",
		})
		return "", false
	}

	return uri.ID, true
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
	case errors.Is(err, model.ErrOrderNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Pesanan tidak ditemukan",
		})
	case errors.Is(err, model.ErrCustomerNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Data customer tidak ditemukan",
		})
	case errors.Is(err, model.ErrInvalidTransition):
		c.JSON(http.StatusConflict, gin.H{
			"status":  "error",
			"message": "Status pesanan tidak bisa diubah ke status tersebut",
		})
	case errors.Is(err, model.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Anda tidak memiliki akses",
		})
	default:
		log.Println(fallbackMessage+":", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": fallbackMessage,
		})
	}
}

// CreateOrder membuat pesanan baru.
// POST /api/v1/orders
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		return
	}

	var req dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	order, err := h.service.Create(c.Request.Context(), actor, req)
	if err != nil {
		respondError(c, err, "Gagal membuat pesanan")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data":   order,
	})
}

// ListOrders mengambil daftar pesanan.
// GET /api/v1/orders?status=&date_from=&date_to=&page=&limit=
func (h *OrderHandler) ListOrders(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		return
	}

	var query dto.ListOrdersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		respondBadRequest(c, err)
		return
	}

	orders, meta, err := h.service.List(c.Request.Context(), actor, query)
	if err != nil {
		respondError(c, err, "Gagal mengambil daftar pesanan")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   orders,
		"meta":   meta,
	})
}

// GetOrder mengambil detail satu pesanan.
// GET /api/v1/orders/:id
func (h *OrderHandler) GetOrder(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		return
	}

	orderID, ok := getOrderID(c)
	if !ok {
		return
	}

	order, err := h.service.Get(c.Request.Context(), actor, orderID)
	if err != nil {
		respondError(c, err, "Gagal mengambil pesanan")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   order,
	})
}

// UpdateStatus mengubah status pesanan (kasir dan admin).
// PATCH /api/v1/orders/:id/status
func (h *OrderHandler) UpdateStatus(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		return
	}

	orderID, ok := getOrderID(c)
	if !ok {
		return
	}

	var req dto.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	order, err := h.service.UpdateStatus(c.Request.Context(), actor, orderID, req)
	if err != nil {
		respondError(c, err, "Gagal mengubah status pesanan")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   order,
	})
}

// CancelOrder membatalkan pesanan oleh customer (hanya saat menunggu).
// POST /api/v1/orders/:id/cancel
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		return
	}

	orderID, ok := getOrderID(c)
	if !ok {
		return
	}

	var req dto.CancelOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		respondBadRequest(c, err)
		return
	}

	order, err := h.service.Cancel(c.Request.Context(), actor, orderID, req)
	if err != nil {
		respondError(c, err, "Gagal membatalkan pesanan")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   order,
	})
}
