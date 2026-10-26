package dto

type OpenShiftRequest struct {
	OpeningCash float64 `json:"opening_cash" binding:"gte=0"`
	Notes       *string `json:"notes"`
}

type CloseShiftRequest struct {
	ClosingCash float64 `json:"closing_cash" binding:"gte=0"`
	Notes       *string `json:"notes"`
}
