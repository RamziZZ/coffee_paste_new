package dto

// CreateReviewRequest adalah body untuk POST /reviews.
// Hanya customer yang punya pesanan berstatus selesai yang boleh
// mengisi; satu review per pesanan.
type CreateReviewRequest struct {
	OrderID string  `json:"order_id" binding:"required,uuid"`
	Rating  int     `json:"rating" binding:"required,min=1,max=5"`
	Comment *string `json:"comment" binding:"omitempty,max=1000"`
}

// ListReviewsQuery adalah query string untuk GET /reviews (kasir/admin).
type ListReviewsQuery struct {
	UserID    string `form:"user_id"`
	MinRating int    `form:"min_rating"`
	MaxRating int    `form:"max_rating"`
	Page      int    `form:"page"`
	Limit     int    `form:"limit"`
}

type ReviewResponse struct {
	ID           string  `json:"id"`
	OrderID      string  `json:"order_id"`
	OrderCode    string  `json:"order_code"`
	UserID       string  `json:"user_id"`
	CustomerName string  `json:"customer_name"`
	Rating       int     `json:"rating"`
	Comment      *string `json:"comment,omitempty"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// Meta adalah info paginasi.
type Meta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}
