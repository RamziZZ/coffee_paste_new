package model

import "time"

type User struct {
	ID           string    `json:"id"`
	Role         string    `json:"role"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Phone        *string   `json:"phone,omitempty"`
	PhotoURL     *string   `json:"photo_url,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	PendingEmail *string   `json:"pending_email,omitempty"`
	GoogleID     *string   `json:"-"`
}
