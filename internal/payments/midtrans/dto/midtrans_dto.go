package dto

// CreateSnapPaymentRequest adalah body untuk POST /payments/midtrans/snap.
type CreateSnapPaymentRequest struct {
	OrderID string `json:"order_id" binding:"required,uuid"`
}

// SnapPaymentResponse adalah response setelah snap transaction dibuat.
type SnapPaymentResponse struct {
	PaymentID   string `json:"payment_id"`
	OrderID     string `json:"order_id"`
	SnapToken   string `json:"snap_token"`
	RedirectURL string `json:"redirect_url"`
	Amount      int    `json:"amount"`
	Status      string `json:"status"`
}

// MidtransNotificationResponse adalah response untuk webhook Midtrans.
type MidtransNotificationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
