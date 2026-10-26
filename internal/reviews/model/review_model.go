package model

import "errors"

// Role user (enum user_role) — sama dengan modul orders.
const (
	RoleAdmin    = "admin"
	RoleKasir    = "kasir"
	RoleCustomer = "customer"
)

// Status pesanan yang boleh direview.
const StatusSelesai = "selesai"

var (
	ErrReviewNotFound     = errors.New("review tidak ditemukan")
	ErrAlreadyReviewed    = errors.New("pesanan sudah direview")
	ErrOrderNotReviewable = errors.New("pesanan tidak dapat direview")
	ErrForbidden          = errors.New("tidak memiliki akses")
)

// ValidationError dipakai untuk kesalahan input yang pesannya
// aman ditampilkan langsung ke client.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Review adalah satu ulasan pelanggan untuk satu pesanan.
type Review struct {
	ID           string
	OrderID      string
	OrderCode    string
	UserID       string
	CustomerName string
	Rating       int
	Comment      *string
	CreatedAt    string
	UpdatedAt    string
}

// OrderContext adalah hasil load order untuk validasi pembuatan review.
type OrderContext struct {
	Status         string
	CustomerName   *string
	CustomerUserID *string
}

// NewReview adalah data untuk membuat review baru.
type NewReview struct {
	OrderID string
	UserID  string
	Rating  int
	Comment *string
}

// ListFilter adalah filter untuk daftar review.
type ListFilter struct {
	UserID    *string
	MinRating *int
	MaxRating *int
	Limit     int
	Offset    int
}
