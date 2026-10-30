package model

import "errors"

// Tipe notifikasi (harus sama dengan enum di database).
const (
	TypeOrderCreated = "order_created"
	TypeOrderStatus  = "order_status"
	TypePaymentPaid  = "payment_paid"
	TypeStockLow     = "stock_low"
)

var ErrNotificationNotFound = errors.New("notifikasi tidak ditemukan")

// ValidationError dipakai untuk kesalahan input yang pesannya
// aman ditampilkan langsung ke client.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Notification adalah satu baris notifikasi in-app.
type Notification struct {
	ID        string
	UserID    string
	Title     string
	Body      *string
	Type      string
	RefID     *string
	IsRead    bool
	CreatedAt string
}

// NewNotification adalah data untuk membuat satu notifikasi.
type NewNotification struct {
	UserID string
	Title  string
	Body   *string
	Type   string
	RefID  *string
}

// ListFilter adalah filter untuk daftar notifikasi milik satu user.
type ListFilter struct {
	UserID string
	Limit  int
	Offset int
}

// DeviceToken adalah satu baris token FCM milik satu user/device.
type DeviceToken struct {
	ID         string
	UserID     string
	FCMToken   string
	DeviceType string
	CreatedAt  string
	UpdatedAt  string
}
