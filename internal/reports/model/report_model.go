package model

import "errors"

// Periode laporan penjualan.
const (
	PeriodDaily   = "daily"
	PeriodWeekly  = "weekly"
	PeriodMonthly = "monthly"
)

var ErrInvalidPeriod = errors.New("periode laporan tidak valid")

// ValidationError dipakai untuk kesalahan input yang pesannya
// aman ditampilkan langsung ke client.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// SalesReport adalah ringkasan penjualan untuk satu periode.
type SalesReport struct {
	Period          string
	StartDate       string
	EndDate         string
	TotalRevenue    int
	TotalOrders     int
	StatusBreakdown []StatusCount
	TopMenus        []TopMenu
	AvgRating       float64
	ReviewCount     int
	GeneratedAt     string
}

// StatusCount adalah jumlah pesanan per status.
type StatusCount struct {
	Status string
	Count  int
}

// TopMenu adalah menu terlaris dalam periode.
type TopMenu struct {
	Rank     int
	MenuID   string
	MenuName string
	QtySold  int
	Revenue  int
}

// SummaryRow adalah hasil query ringkasan dari database.
type SummaryRow struct {
	TotalRevenue int
	TotalOrders  int
	Menunggu     int
	Proses       int
	Siap         int
	Selesai      int
	Dibatalkan   int
}

// RatingRow adalah hasil query rata-rata rating dari database.
type RatingRow struct {
	AvgRating   float64
	ReviewCount int
}
