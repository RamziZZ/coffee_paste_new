package dto

type CreateOrderItemRequest struct {
	MenuID   string  `json:"menu_id" binding:"required"`
	Qty      int     `json:"qty" binding:"required,min=1"`
	ItemNote *string `json:"item_note"`
}

type UpdateOrderItemRequest struct {
	Qty      int     `json:"qty" binding:"required,min=1"`
	ItemNote *string `json:"item_note"`
}
