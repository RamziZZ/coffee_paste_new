package model

import "time"

type Shift struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	OpeningCash    float64    `json:"opening_cash"`
	ClosingCash    *float64   `json:"closing_cash,omitempty"`
	ExpectedCash   *float64   `json:"expected_cash,omitempty"`
	CashDifference *float64   `json:"cash_difference,omitempty"`
	Status         string     `json:"status"`
	OpenedAt       time.Time  `json:"opened_at"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
