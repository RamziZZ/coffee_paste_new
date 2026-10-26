package pdf

import (
	"testing"

	"coffee_paste_backend/internal/reports/model"
)

func TestFormatRupiah(t *testing.T) {
	cases := map[int]string{
		0:        "Rp 0",
		5:        "Rp 5",
		1234:     "Rp 1.234",
		1234567:  "Rp 1.234.567",
		-1234567: "Rp -1.234.567",
	}
	for input, want := range cases {
		if got := FormatRupiah(input); got != want {
			t.Errorf("FormatRupiah(%d) = %q, want %q", input, got, want)
		}
	}
}

func TestGenerateSalesReport(t *testing.T) {
	report := &model.SalesReport{
		Period:       model.PeriodDaily,
		StartDate:    "2026-10-12",
		EndDate:      "2026-10-12",
		TotalRevenue: 1234567,
		TotalOrders:  42,
		StatusBreakdown: []model.StatusCount{
			{Status: "selesai", Count: 30},
			{Status: "dibatalkan", Count: 2},
		},
		TopMenus: []model.TopMenu{
			{Rank: 1, MenuName: "Kopi Susu", QtySold: 25, Revenue: 375000},
		},
		AvgRating:   4.5,
		ReviewCount: 12,
		GeneratedAt: "2026-10-12 15:04 WIB",
	}

	content, err := GenerateSalesReport(report)
	if err != nil {
		t.Fatalf("GenerateSalesReport error: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("PDF kosong")
	}
	if string(content[:4]) != "%PDF" {
		t.Fatalf("Bukan file PDF valid: %q", content[:4])
	}
}
