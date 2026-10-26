package model

import (
	"time"

	"github.com/google/uuid"
)

type OrderItem struct {
	ID       uuid.UUID `json:"id"`
	OrderID  uuid.UUID `json:"order_id"`
	MenuID   uuid.UUID `json:"menu_id"`
	MenuName string    `json:"menu_name"`
	Price    int       `json:"price"`
	Qty      int       `json:"qty"`
	ItemNote *string   `json:"item_note"`
}

type OrderItemResponse struct {
	Item      OrderItem  `json:"item"`
	Subtotal  int        `json:"subtotal"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}
