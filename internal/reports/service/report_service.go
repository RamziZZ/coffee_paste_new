package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"coffee_paste_backend/internal/reports/dto"
	"coffee_paste_backend/internal/reports/model"
	"coffee_paste_backend/internal/reports/pdf"
	"coffee_paste_backend/internal/reports/repository"
)

// topMenuLimit adalah jumlah menu teratas yang ditampilkan laporan.
const topMenuLimit = 5

type ReportService struct {
	repository *repository.ReportRepository
}

func NewReportService(
	repository *repository.ReportRepository,
) *ReportService {
	return &ReportService{
		repository: repository,
	}
}

// GetSalesReport membangun laporan penjualan harian/mingguan/bulanan.
func (s *ReportService) GetSalesReport(
	ctx context.Context,
	q dto.SalesReportQuery,
) (*model.SalesReport, error) {

	start, end, err := resolveRange(q.Period, q.Date)
	if err != nil {
		return nil, err
	}

	summary, err := s.repository.GetSalesSummary(ctx, start, end)
	if err != nil {
		return nil, err
	}

	topMenus, err := s.repository.GetTopMenus(ctx, start, end, topMenuLimit)
	if err != nil {
		return nil, err
	}

	rating, err := s.repository.GetRatingSummary(ctx, start, end)
	if err != nil {
		return nil, err
	}

	report := &model.SalesReport{
		Period:       q.Period,
		StartDate:    start.Format("2006-01-02"),
		EndDate:      end.AddDate(0, 0, -1).Format("2006-01-02"),
		TotalRevenue: summary.TotalRevenue,
		TotalOrders:  summary.TotalOrders,
		StatusBreakdown: []model.StatusCount{
			{Status: "menunggu", Count: summary.Menunggu},
			{Status: "proses", Count: summary.Proses},
			{Status: "siap", Count: summary.Siap},
			{Status: "selesai", Count: summary.Selesai},
			{Status: "dibatalkan", Count: summary.Dibatalkan},
		},
		TopMenus:    topMenus,
		AvgRating:   rating.AvgRating,
		ReviewCount: rating.ReviewCount,
		GeneratedAt: pdf.NowJakarta(),
	}

	return report, nil
}

// BuildSalesReportPDF membangun file PDF laporan penjualan.
// Mengembalikan isi PDF dan nama file unduhan.
func (s *ReportService) BuildSalesReportPDF(
	ctx context.Context,
	q dto.SalesReportQuery,
) ([]byte, string, error) {

	report, err := s.GetSalesReport(ctx, q)
	if err != nil {
		return nil, "", err
	}

	content, err := pdf.GenerateSalesReport(report)
	if err != nil {
		return nil, "", fmt.Errorf("gagal membuat PDF: %w", err)
	}

	filename := fmt.Sprintf(
		"laporan-penjualan-%s-%s.pdf",
		q.Period, report.EndDate,
	)

	return content, filename, nil
}

// resolveRange menentukan rentang waktu [start, end) dari periode dan
// tanggal acuan (kosong = hari ini). Tanggal dibangun di zona waktu
// lokal server.
func resolveRange(
	period, dateStr string,
) (time.Time, time.Time, error) {

	dateStr = strings.TrimSpace(dateStr)

	var base time.Time
	if dateStr == "" {
		now := time.Now()
		base = time.Date(
			now.Year(), now.Month(), now.Day(),
			0, 0, 0, 0, now.Location(),
		)
	} else {
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return time.Time{}, time.Time{}, &model.ValidationError{
				Message: "Format tanggal harus YYYY-MM-DD",
			}
		}
		base = parsed
	}

	switch period {
	case model.PeriodDaily:
		return base, base.AddDate(0, 0, 1), nil

	case model.PeriodWeekly:
		// Senin = hari awal minggu.
		weekday := int(base.Weekday())
		mondayOffset := (weekday + 6) % 7
		start := base.AddDate(0, 0, -mondayOffset)
		return start, start.AddDate(0, 0, 7), nil

	case model.PeriodMonthly:
		start := time.Date(
			base.Year(), base.Month(), 1,
			0, 0, 0, 0, base.Location(),
		)
		return start, start.AddDate(0, 1, 0), nil

	default:
		return time.Time{}, time.Time{}, &model.ValidationError{
			Message: "Periode harus daily, weekly, atau monthly",
		}
	}
}
