package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"coffee_paste_backend/internal/reviews/dto"
	"coffee_paste_backend/internal/reviews/model"
	"coffee_paste_backend/internal/reviews/service"
)

type ReviewHandler struct {
	service *service.ReviewService
}

func NewReviewHandler(service *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{
		service: service,
	}
}

// reviewURI memvalidasi parameter :id di URL harus berupa UUID.
type reviewURI struct {
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

	userID, ok := userValue.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Data user tidak valid",
		})
		return service.Actor{}, false
	}

	role, _ := roleValue.(string)
	if role == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Data user tidak valid",
		})
		return service.Actor{}, false
	}

	return service.Actor{UserID: userID, Role: role}, true
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
	case errors.Is(err, model.ErrReviewNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Review tidak ditemukan",
		})
	case errors.Is(err, model.ErrAlreadyReviewed):
		c.JSON(http.StatusConflict, gin.H{
			"status":  "error",
			"message": "Pesanan sudah pernah direview",
		})
	case errors.Is(err, model.ErrOrderNotReviewable):
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Pesanan tidak dapat direview",
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

// Create membuat review oleh customer untuk pesanannya yang selesai.
// POST /api/v1/reviews
func (h *ReviewHandler) Create(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		return
	}

	var req dto.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	review, err := h.service.Create(c.Request.Context(), actor, req)
	if err != nil {
		respondError(c, err, "Gagal membuat review")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data":   review,
	})
}

// Get mengambil detail review.
// GET /api/v1/reviews/:id
func (h *ReviewHandler) Get(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		return
	}

	var uri reviewURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Review ID tidak valid",
		})
		return
	}

	review, err := h.service.Get(c.Request.Context(), actor, uri.ID)
	if err != nil {
		respondError(c, err, "Gagal mengambil review")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   review,
	})
}

// ListMine mengambil daftar review milik customer yang login.
// GET /api/v1/reviews/mine?page=&limit=
func (h *ReviewHandler) ListMine(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		return
	}

	var query struct {
		Page  int `form:"page"`
		Limit int `form:"limit"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		respondBadRequest(c, err)
		return
	}

	reviews, meta, err := h.service.ListMine(
		c.Request.Context(), actor, query.Page, query.Limit,
	)
	if err != nil {
		respondError(c, err, "Gagal mengambil daftar review")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   reviews,
		"meta":   meta,
	})
}

// ListAll mengambil daftar review untuk staf dengan filter.
// GET /api/v1/reviews?user_id=&min_rating=&max_rating=&page=&limit=
func (h *ReviewHandler) ListAll(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		return
	}

	var query dto.ListReviewsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		respondBadRequest(c, err)
		return
	}

	reviews, meta, err := h.service.ListAll(c.Request.Context(), actor, query)
	if err != nil {
		respondError(c, err, "Gagal mengambil daftar review")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   reviews,
		"meta":   meta,
	})
}

// Delete menghapus review (moderasi admin).
// DELETE /api/v1/reviews/:id
func (h *ReviewHandler) Delete(c *gin.Context) {
	actor, ok := getActor(c)
	if !ok {
		return
	}

	var uri reviewURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Review ID tidak valid",
		})
		return
	}

	if err := h.service.Delete(c.Request.Context(), actor, uri.ID); err != nil {
		respondError(c, err, "Gagal menghapus review")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Review berhasil dihapus",
	})
}
