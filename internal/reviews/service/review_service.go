package service

import (
	"context"
	"strings"

	"coffee_paste_backend/internal/reviews/dto"
	"coffee_paste_backend/internal/reviews/model"
	"coffee_paste_backend/internal/reviews/repository"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// Actor adalah user yang sedang login (dari JWT).
type Actor struct {
	UserID string
	Role   string
}

func (a Actor) isStaff() bool {
	return a.Role == model.RoleKasir || a.Role == model.RoleAdmin
}

type ReviewService struct {
	repository *repository.ReviewRepository
}

func NewReviewService(
	repository *repository.ReviewRepository,
) *ReviewService {
	return &ReviewService{
		repository: repository,
	}
}

// cleanPtr merapikan string opsional: kosong menjadi nil.
func cleanPtr(s *string) *string {
	if s == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func toReviewResponse(r model.Review) dto.ReviewResponse {
	return dto.ReviewResponse{
		ID:           r.ID,
		OrderID:      r.OrderID,
		OrderCode:    r.OrderCode,
		UserID:       r.UserID,
		CustomerName: r.CustomerName,
		Rating:       r.Rating,
		Comment:      r.Comment,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

// Create membuat review oleh customer untuk pesanannya yang selesai.
// Satu review per pesanan.
func (s *ReviewService) Create(
	ctx context.Context,
	actor Actor,
	req dto.CreateReviewRequest,
) (*dto.ReviewResponse, error) {

	if actor.Role != model.RoleCustomer {
		return nil, model.ErrForbidden
	}

	orderID := strings.ToLower(strings.TrimSpace(req.OrderID))

	oc, err := s.repository.LoadOrderContext(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if oc.CustomerUserID == nil ||
		!strings.EqualFold(*oc.CustomerUserID, actor.UserID) {
		// Pesanan orang lain dianggap tidak bisa direview.
		return nil, model.ErrOrderNotReviewable
	}

	if oc.Status != model.StatusSelesai {
		return nil, &model.ValidationError{
			Message: "Hanya pesanan yang selesai yang bisa direview",
		}
	}

	review, err := s.repository.Create(ctx, model.NewReview{
		OrderID: orderID,
		UserID:  actor.UserID,
		Rating:  req.Rating,
		Comment: cleanPtr(req.Comment),
	})
	if err != nil {
		return nil, err
	}

	response := toReviewResponse(*review)
	return &response, nil
}

// Get mengambil detail review. Customer hanya review miliknya sendiri;
// staf boleh melihat semua.
func (s *ReviewService) Get(
	ctx context.Context,
	actor Actor,
	id string,
) (*dto.ReviewResponse, error) {

	review, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !actor.isStaff() && review.UserID != actor.UserID {
		return nil, model.ErrReviewNotFound
	}

	response := toReviewResponse(*review)
	return &response, nil
}

// ListMine mengambil daftar review milik customer yang login.
func (s *ReviewService) ListMine(
	ctx context.Context,
	actor Actor,
	page, limit int,
) ([]dto.ReviewResponse, dto.Meta, error) {

	if actor.Role != model.RoleCustomer {
		return nil, dto.Meta{}, model.ErrForbidden
	}

	page, limit = normalizePageLimit(page, limit)

	userID := actor.UserID

	reviews, total, err := s.repository.List(ctx, model.ListFilter{
		UserID: &userID,
		Limit:  limit,
		Offset: (page - 1) * limit,
	})
	if err != nil {
		return nil, dto.Meta{}, err
	}

	result := make([]dto.ReviewResponse, 0, len(reviews))
	for _, r := range reviews {
		result = append(result, toReviewResponse(r))
	}

	return result, dto.Meta{Page: page, Limit: limit, Total: total}, nil
}

// ListAll mengambil daftar review untuk staf (kasir/admin) dengan filter.
func (s *ReviewService) ListAll(
	ctx context.Context,
	actor Actor,
	q dto.ListReviewsQuery,
) ([]dto.ReviewResponse, dto.Meta, error) {

	if !actor.isStaff() {
		return nil, dto.Meta{}, model.ErrForbidden
	}

	page, limit := normalizePageLimit(q.Page, q.Limit)

	filter := model.ListFilter{
		Limit:  limit,
		Offset: (page - 1) * limit,
	}

	if userID := strings.TrimSpace(q.UserID); userID != "" {
		filter.UserID = &userID
	}
	if q.MinRating > 0 {
		if q.MinRating > 5 {
			return nil, dto.Meta{}, &model.ValidationError{
				Message: "min_rating harus antara 1 dan 5",
			}
		}
		filter.MinRating = &q.MinRating
	}
	if q.MaxRating > 0 {
		if q.MaxRating > 5 {
			return nil, dto.Meta{}, &model.ValidationError{
				Message: "max_rating harus antara 1 dan 5",
			}
		}
		if filter.MinRating != nil && q.MaxRating < *filter.MinRating {
			return nil, dto.Meta{}, &model.ValidationError{
				Message: "max_rating tidak boleh kurang dari min_rating",
			}
		}
		filter.MaxRating = &q.MaxRating
	}

	reviews, total, err := s.repository.List(ctx, filter)
	if err != nil {
		return nil, dto.Meta{}, err
	}

	result := make([]dto.ReviewResponse, 0, len(reviews))
	for _, r := range reviews {
		result = append(result, toReviewResponse(r))
	}

	return result, dto.Meta{Page: page, Limit: limit, Total: total}, nil
}

// Delete menghapus review (hanya admin, moderasi).
func (s *ReviewService) Delete(
	ctx context.Context,
	actor Actor,
	id string,
) error {

	if actor.Role != model.RoleAdmin {
		return model.ErrForbidden
	}

	return s.repository.Delete(ctx, id)
}

func normalizePageLimit(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}
