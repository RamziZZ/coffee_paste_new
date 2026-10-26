package model

import (
	"errors"
	"time"
)

// Role user (enum user_role).
const (
	RoleAdmin    = "admin"
	RoleKasir    = "kasir"
	RoleCustomer = "customer"
)

// Status pesanan (enum order_status).
const (
	StatusMenunggu   = "menunggu"
	StatusProses     = "proses"
	StatusSiap       = "siap"
	StatusSelesai    = "selesai"
	StatusDibatalkan = "dibatalkan"
)

// Sumber pesanan (enum order_source).
const (
	SourceKasir       = "kasir"
	SourceCustomerApp = "customer_app"
)

// Tipe pesanan (enum order_type).
const (
	TypeDineIn   = "dine_in"
	TypeTakeAway = "take_away"
)

// Status menu yang boleh dipesan (enum menu_status).
const MenuStatusTersedia = "tersedia"

var (
	ErrOrderNotFound     = errors.New("pesanan tidak ditemukan")
	ErrCustomerNotFound  = errors.New("customer tidak ditemukan")
	ErrInvalidTransition = errors.New("perubahan status tidak diperbolehkan")
	ErrForbidden         = errors.New("tidak memiliki akses")
)

// ValidationError dipakai untuk kesalahan input yang pesannya
// aman ditampilkan langsung ke client.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Alur status pesanan. selesai dan dibatalkan adalah status akhir.
var allowedTransitions = map[string][]string{
	StatusMenunggu: {StatusProses, StatusDibatalkan},
	StatusProses:   {StatusSiap, StatusDibatalkan},
	StatusSiap:     {StatusSelesai, StatusDibatalkan},
}

// CanTransition memeriksa apakah status boleh berpindah dari -> ke.
func CanTransition(from, to string) bool {
	for _, next := range allowedTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

// Order adalah satu baris orders beserta subtotal hasil hitung
// dari order_items dan data customer (kalau ada).
type Order struct {
	ID             string
	OrderCode      string
	CustomerID     *string
	CustomerUserID *string
	CustomerName   *string
	CashierID      *string
	TableNumber    *string
	Source         string
	OrderType      string
	Status         string
	Subtotal       int
	Discount       int
	Note           *string
	CancelReason   *string
	ItemCount      int
	OrderedAt      time.Time
}

// Total adalah subtotal dikurangi diskon (tidak pernah negatif).
func (o Order) Total() int {
	total := o.Subtotal - o.Discount
	if total < 0 {
		return 0
	}
	return total
}

type OrderItem struct {
	ID       string
	MenuID   string
	MenuName string
	Price    int
	Qty      int
	ItemNote *string
}

type StatusLog struct {
	Status        string
	Note          *string
	ChangedBy     string
	ChangedByName string
	ChangedAt     time.Time
}

// NewItem adalah satu baris pesanan dari client (harga diambil dari database).
type NewItem struct {
	MenuID   string
	Qty      int
	ItemNote *string
}

// NewOrder adalah data lengkap untuk membuat pesanan baru.
type NewOrder struct {
	CustomerID  *string
	CashierID   *string
	TableNumber *string
	Source      string
	OrderType   string
	Discount    int
	Note        *string
	ChangedBy   string
	Items       []NewItem
}

// StatusChange adalah permintaan memindahkan status pesanan.
// RequireCurrent (opsional) memastikan status saat ini sesuai,
// supaya pengecekan dan perubahan berlangsung atomik.
type StatusChange struct {
	OrderID        string
	NewStatus      string
	Note           *string
	CancelReason   *string
	ChangedBy      string
	RequireCurrent string
}

// ListFilter adalah filter untuk daftar pesanan.
type ListFilter struct {
	CustomerUserID *string
	Status         *string
	DateFrom       *string
	DateTo         *string
	Limit          int
	Offset         int
}
