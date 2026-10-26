package model

import "time"

// Favorite adalah baris favorite_menus yang digabung dengan data menus
// sehingga response sudah membawa info menu.
type Favorite struct {
	MenuID    string    `json:"menu_id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Price     int       `json:"price"`
	ImageURL  *string   `json:"image_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
