package dto

// ListNotificationsQuery adalah query string untuk GET /notifications.
type ListNotificationsQuery struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

type NotificationResponse struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Body      *string `json:"body,omitempty"`
	Type      string  `json:"type"`
	RefID     *string `json:"ref_id,omitempty"`
	IsRead    bool    `json:"is_read"`
	CreatedAt string  `json:"created_at"`
}

// Meta adalah info paginasi plus jumlah notifikasi belum dibaca.
type Meta struct {
	Page        int `json:"page"`
	Limit       int `json:"limit"`
	Total       int `json:"total"`
	UnreadCount int `json:"unread_count"`
}
