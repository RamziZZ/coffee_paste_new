package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"coffee_paste_backend/internal/notifications/dto"
	"coffee_paste_backend/internal/notifications/model"
	"coffee_paste_backend/internal/notifications/service"
)

type NotificationHandler struct {
	service *service.NotificationService
}

func NewNotificationHandler(service *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		service: service,
	}
}

// readURI memvalidasi parameter :id di URL harus berupa UUID.
type readURI struct {
	ID string `uri:"id" binding:"required,uuid"`
}

// getUserID mengambil user_id yang diset oleh AuthMiddleware.
func getUserID(c *gin.Context) (string, bool) {
	userValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Data user tidak ditemukan",
		})
		return "", false
	}

	userID, ok := userValue.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Data user tidak valid",
		})
		return "", false
	}

	return userID, true
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
	case errors.Is(err, model.ErrNotificationNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Notifikasi tidak ditemukan",
		})
	default:
		log.Println(fallbackMessage+":", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": fallbackMessage,
		})
	}
}

// List mengambil daftar notifikasi milik user yang login.
// GET /api/v1/notifications?page=&limit=
func (h *NotificationHandler) List(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var query dto.ListNotificationsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Data request tidak valid",
			"detail":  err.Error(),
		})
		return
	}

	notifications, meta, err := h.service.List(
		c.Request.Context(), userID, query,
	)
	if err != nil {
		respondError(c, err, "Gagal mengambil daftar notifikasi")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   notifications,
		"meta":   meta,
	})
}

// UnreadCount menghitung notifikasi yang belum dibaca.
// GET /api/v1/notifications/unread-count
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	count, err := h.service.UnreadCount(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err, "Gagal menghitung notifikasi")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"unread_count": count,
		},
	})
}

// MarkRead menandai satu notifikasi sebagai sudah dibaca.
// PATCH /api/v1/notifications/:id/read
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var uri readURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Notifikasi ID tidak valid",
		})
		return
	}

	if err := h.service.MarkRead(
		c.Request.Context(), userID, uri.ID,
	); err != nil {
		respondError(c, err, "Gagal menandai notifikasi dibaca")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Notifikasi ditandai sudah dibaca",
	})
}

// MarkAllRead menandai semua notifikasi sebagai sudah dibaca.
// PATCH /api/v1/notifications/read-all
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	if err := h.service.MarkAllRead(c.Request.Context(), userID); err != nil {
		respondError(c, err, "Gagal menandai semua notifikasi dibaca")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Semua notifikasi ditandai sudah dibaca",
	})
}

// RegisterDeviceToken mendaftarkan token FCM untuk push notification.
// POST /api/v1/notifications/devices
func (h *NotificationHandler) RegisterDeviceToken(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req dto.RegisterDeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Data device token tidak valid",
		})
		return
	}

	if err := h.service.RegisterDeviceToken(
		c.Request.Context(), userID, req,
	); err != nil {
		respondError(c, err, "Gagal mendaftarkan device token")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Device token berhasil didaftarkan",
	})
}

// DeleteDeviceToken menghapus token FCM (unregister push notification).
// DELETE /api/v1/notifications/devices
func (h *NotificationHandler) DeleteDeviceToken(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req dto.DeleteDeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Data device token tidak valid",
		})
		return
	}

	if err := h.service.DeleteDeviceToken(
		c.Request.Context(), userID, req.FCMToken,
	); err != nil {
		respondError(c, err, "Gagal menghapus device token")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Device token berhasil dihapus",
	})
}
