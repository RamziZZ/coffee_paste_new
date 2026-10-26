package dto

import "time"

// CreateOrderItemRequest adalah satu baris menu dalam pesanan.
type CreateOrderItemRequest struct {
	MenuID   string  `json:"menu_id" binding:"required,uuid"`
	Qty      int     `json:"qty" binding:"required,min=1,max=99"`
	ItemNote *string `json:"item_note" binding:"omitempty,max=100"`
}

// CreateOrderRequest adalah body untuk POST /orders.
// customer_id dan discount hanya dipakai kalau pembuatnya kasir atau admin.
type CreateOrderRequest struct {
	CustomerID  *string                  `json:"customer_id" binding:"omitempty,uuid"`
	TableNumber *string                  `json:"table_number" binding:"omitempty,max=20"`
	OrderType   string                   `json:"order_type" binding:"required,oneof=dine_in take_away"`
	Discount    int                      `json:"discount" binding:"min=0"`
	Note        *string                  `json:"note" binding:"omitempty,max=500"`
	Items       []CreateOrderItemRequest `json:"items" binding:"required,min=1,max=50,dive"`
}

// UpdateStatusRequest adalah body untuk PATCH /orders/:id/status.
type UpdateStatusRequest struct {
	Status       string  `json:"status" binding:"required,oneof=proses siap selesai dibatalkan"`
	Note         *string `json:"note" binding:"omitempty,max=255"`
	CancelReason *string `json:"cancel_reason" binding:"omitempty,max=500"`
}

// CancelOrderRequest adalah body (opsional) untuk POST /orders/:id/cancel.
type CancelOrderRequest struct {
	Reason *string `json:"reason" binding:"omitempty,max=500"`
}

// ListOrdersQuery adalah query string untuk GET /orders.
type ListOrdersQuery struct {
	Status   string `form:"status"`
	DateFrom string `form:"date_from"`
	DateTo   string `form:"date_to"`
	Page     int    `form:"page"`
	Limit    int    `form:"limit"`
}

type OrderItemResponse struct {
	ID       string  `json:"id"`
	MenuID   string  `json:"menu_id"`
	MenuName string  `json:"menu_name"`
	Price    int     `json:"price"`
	Qty      int     `json:"qty"`
	ItemNote *string `json:"item_note"`
	Subtotal int     `json:"subtotal"`
}

type StatusLogResponse struct {
	Status        string    `json:"status"`
	Note          *string   `json:"note"`
	ChangedBy     string    `json:"changed_by"`
	ChangedByName string    `json:"changed_by_name"`
	ChangedAt     time.Time `json:"changed_at"`
}

// OrderResponse dipakai untuk daftar (tanpa items dan status_logs)
// maupun detail pesanan (dengan items dan status_logs).
type OrderResponse struct {
	ID           string              `json:"id"`
	OrderCode    string              `json:"order_code"`
	CustomerID   *string             `json:"customer_id"`
	CustomerName *string             `json:"customer_name"`
	CashierID    *string             `json:"cashier_id"`
	TableNumber  *string             `json:"table_number"`
	Source       string              `json:"source"`
	OrderType    string              `json:"order_type"`
	Status       string              `json:"status"`
	Subtotal     int                 `json:"subtotal"`
	Discount     int                 `json:"discount"`
	Total        int                 `json:"total"`
	Note         *string             `json:"note"`
	CancelReason *string             `json:"cancel_reason"`
	ItemCount    int                 `json:"item_count"`
	OrderedAt    time.Time           `json:"ordered_at"`
	Items        []OrderItemResponse `json:"items,omitempty"`
	StatusLogs   []StatusLogResponse `json:"status_logs,omitempty"`
}

// Meta adalah info paginasi.
type Meta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}
