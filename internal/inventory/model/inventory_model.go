package model

import "errors"

var (
	ErrStockNotFound     = errors.New("stok menu tidak ditemukan")
	ErrInsufficientStock = errors.New("stok tidak mencukupi")
	ErrMenuNotFound      = errors.New("menu tidak ditemukan")
)

// ValidationError dipakai untuk kesalahan input yang pesannya
// aman ditampilkan langsung ke client.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// StockRecord adalah satu baris stok menu beserta nama menu.
type StockRecord struct {
	MenuID            string
	MenuName          string
	Qty               int
	LowStockThreshold int
	UpdatedAt         string
}

// LowStockItem adalah menu yang stoknya sudah di bawah threshold.
type LowStockItem struct {
	MenuID    string
	MenuName  string
	Qty       int
	Threshold int
}

// ListFilter adalah filter untuk daftar stok.
type ListFilter struct {
	LowOnly bool
	Limit   int
	Offset  int
}
