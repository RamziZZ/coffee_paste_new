package dto

// ListInventoryQuery adalah query string untuk GET /inventory.
type ListInventoryQuery struct {
	LowStock bool `form:"low_stock"`
	Page     int  `form:"page"`
	Limit    int  `form:"limit"`
}

// AdjustStockRequest adalah body untuk POST /inventory/adjust.
// Salah satu dari change (inkremental) atau set_qty (set langsung)
// wajib diisi. low_stock_threshold bersifat opsional.
type AdjustStockRequest struct {
	MenuID            string `json:"menu_id" binding:"required,uuid"`
	Change            *int   `json:"change" binding:"omitempty"`
	SetQty            *int   `json:"set_qty" binding:"omitempty,min=0"`
	LowStockThreshold *int   `json:"low_stock_threshold" binding:"omitempty,min=0"`
}

type StockResponse struct {
	MenuID            string `json:"menu_id"`
	MenuName          string `json:"menu_name"`
	Qty               int    `json:"qty"`
	LowStockThreshold int    `json:"low_stock_threshold"`
	IsLowStock        bool   `json:"is_low_stock"`
}

// Meta adalah info paginasi.
type Meta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}
