package model

import "time"

// MidtransPayment menyimpan status pembayaran melalui Midtrans.
type MidtransPayment struct {
	ID              string     `json:"id"`
	OrderID         string     `json:"order_id"`
	MidtransTransID *string    `json:"midtrans_trans_id,omitempty"`
	SnapToken       *string    `json:"snap_token,omitempty"`
	RedirectURL     *string    `json:"redirect_url,omitempty"`
	Amount          int        `json:"amount"`
	Status          string     `json:"status"`
	PaymentType     *string    `json:"payment_type,omitempty"`
	Bank            *string    `json:"bank,omitempty"`
	MaskedCard      *string    `json:"masked_card,omitempty"`
	FraudStatus     *string    `json:"fraud_status,omitempty"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
