package dto

type MenuFilterResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	FilterLabel string `json:"filter_label"`
	Icon        string `json:"icon,omitempty"`
}

type CreateMenuRequest struct {
	CategoryID  string  `json:"category_id" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	Price       int     `json:"price" binding:"gte=0"`
	Status      string  `json:"status" binding:"required"`
	ImageURL    *string `json:"image_url"`
	IsFeatured  bool    `json:"is_featured"`
}

type UpdateMenuRequest struct {
	CategoryID  string  `json:"category_id" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	Price       int     `json:"price" binding:"gte=0"`
	Status      string  `json:"status" binding:"required"`
	ImageURL    *string `json:"image_url"`
	IsFeatured  bool    `json:"is_featured"`
}
