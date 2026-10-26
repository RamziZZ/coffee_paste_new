package model

import "time"

type Payment struct {
	ID           string     `json:"id"`
	OrderID      string     `json:"order_id"`
	ReceivedBy   string     `json:"received_by"`
	Method       string     `json:"method"`
	Amount       int        `json:"amount"`
	CashReceived *int       `json:"cash_received,omitempty"`
	ChangeAmount *int       `json:"change_amount,omitempty"`
	Status       string     `json:"status"`
	ReferenceNo  *string    `json:"reference_no,omitempty"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`
}
