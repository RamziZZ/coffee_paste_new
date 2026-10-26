package handler

import (
	"errors"
	"net/http"

	"coffee_paste_backend/internal/shifts/dto"
	"coffee_paste_backend/internal/shifts/repository"
	"coffee_paste_backend/internal/shifts/service"

	"github.com/gin-gonic/gin"
)

type ShiftHandler struct {
	service *service.ShiftService
}

func NewShiftHandler(svc *service.ShiftService) *ShiftHandler {
	return &ShiftHandler{
		service: svc,
	}
}

func currentUserID(c *gin.Context) (string, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		return "", false
	}

	userID, ok := value.(string)
	if !ok || userID == "" {
		return "", false
	}

	return userID, true
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrShiftNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Shift tidak ditemukan",
		})

	case errors.Is(err, repository.ErrShiftAlreadyOpen):
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "Kamu masih memiliki shift aktif",
		})

	case errors.Is(err, repository.ErrShiftAlreadyClosed):
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "Shift sudah ditutup",
		})

	case errors.Is(err, service.ErrInvalidShiftData):
		c.JSON(http.StatusBadRequest, gin.H{
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

// POST /api/v1/shifts
func (h *ShiftHandler) Open(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Identitas pengguna tidak ditemukan",
		})
		return
	}

	var req dto.OpenShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Data pembukaan shift tidak valid",
		})
		return
	}

	shift, err := h.service.Open(c.Request.Context(), userID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Shift berhasil dibuka",
		"data":    shift,
	})
}

// GET /api/v1/shifts/active
func (h *ShiftHandler) GetActive(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Identitas pengguna tidak ditemukan",
		})
		return
	}

	shift, err := h.service.GetActive(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Shift aktif berhasil diambil",
		"data":    shift,
	})
}

// GET /api/v1/shifts
func (h *ShiftHandler) ListMine(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Identitas pengguna tidak ditemukan",
		})
		return
	}

	shifts, err := h.service.ListMine(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Riwayat shift berhasil diambil",
		"data":    shifts,
	})
}

// PATCH /api/v1/shifts/:id/close
func (h *ShiftHandler) Close(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Identitas pengguna tidak ditemukan",
		})
		return
	}

	shiftID := c.Param("id")

	var req dto.CloseShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Data penutupan shift tidak valid",
		})
		return
	}

	shift, err := h.service.Close(
		c.Request.Context(),
		shiftID,
		userID,
		&req,
	)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Shift berhasil ditutup",
		"data":    shift,
	})
}
