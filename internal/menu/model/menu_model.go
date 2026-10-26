package model

type MenuCategory struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type MenuItem struct {
	ID          string  `json:"id"`
	CategoryID  string  `json:"category_id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Price       int     `json:"price"`
	Status      string  `json:"status"`
	ImageURL    *string `json:"image_url,omitempty"`
	IsFeatured  bool    `json:"is_featured"`
}
