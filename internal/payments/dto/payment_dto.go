package dto

type CreatePaymentRequest struct {
	OrderID      string  `json:"order_id" binding:"required"`
	Method       string  `json:"method" binding:"required,oneof=tunai qris"`
	Amount       int     `json:"amount" binding:"required,gt=0"`
	CashReceived *int    `json:"cash_received"`
	ReferenceNo  *string `json:"reference_no"`
}

type UpdatePaymentStatusRequest struct {
	Status       string `json:"status" binding:"required,oneof=pending success failed expired refunded"`
	CashReceived *int   `json:"cash_received"`
}
