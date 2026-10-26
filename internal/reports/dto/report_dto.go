package dto

// SalesReportQuery adalah query string untuk endpoint laporan.
// date kosong berarti hari ini; format YYYY-MM-DD.
type SalesReportQuery struct {
	Period string `form:"period" binding:"required,oneof=daily weekly monthly"`
	Date   string `form:"date" binding:"omitempty"`
}
