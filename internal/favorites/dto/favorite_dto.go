package dto

type FavoriteResponse struct {
	MenuID   string  `json:"menu_id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Price    int     `json:"price"`
	ImageURL *string `json:"image_url,omitempty"`
}
