// Package pdf berisi generator PDF untuk ekspor laporan penjualan.
package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"

	"coffee_paste_backend/internal/reports/model"
)

// GenerateSalesReport membuat file PDF laporan penjualan.
func GenerateSalesReport(report *model.SalesReport) ([]byte, error) {
	p := fpdf.New("P", "mm", "A4", "")
	p.SetMargins(15, 15, 15)
	p.SetAutoPageBreak(true, 20)
	p.AddPage()

	// Kop laporan
	p.SetFont("Helvetica", "B", 16)
	p.CellFormat(0, 8, "Coffee Paste", "", 1, "L", false, 0, "")
	p.SetFont("Helvetica", "B", 13)
	p.CellFormat(0, 8, "Laporan Penjualan", "", 1, "L", false, 0, "")

	p.SetFont("Helvetica", "", 10)
	p.CellFormat(0, 6, "Periode: "+periodLabel(report.Period), "", 1, "L", false, 0, "")
	p.CellFormat(0, 6,
		fmt.Sprintf("Tanggal: %s s/d %s", report.StartDate, report.EndDate),
		"", 1, "L", false, 0, "",
	)
	p.CellFormat(0, 6,
		"Dicetak: "+report.GeneratedAt,
		"", 1, "L", false, 0, "",
	)
	p.Ln(4)

	// Ringkasan
	sectionTitle(p, "Ringkasan")
	drawSummaryBox(p, report)
	p.Ln(6)

	// Pesanan per status
	sectionTitle(p, "Pesanan per Status")
	drawStatusTable(p, report.StatusBreakdown)
	p.Ln(6)

	// Top menu
	sectionTitle(p, "Menu Terlaris")
	drawTopMenuTable(p, report.TopMenus)

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func periodLabel(period string) string {
	switch period {
	case model.PeriodDaily:
		return "Harian"
	case model.PeriodWeekly:
		return "Mingguan"
	case model.PeriodMonthly:
		return "Bulanan"
	default:
		return period
	}
}

func sectionTitle(p *fpdf.Fpdf, title string) {
	p.SetFont("Helvetica", "B", 12)
	p.SetTextColor(30, 30, 30)
	p.CellFormat(0, 8, title, "", 1, "L", false, 0, "")

	// Garis bawah judul section
	y := p.GetY()
	p.SetDrawColor(180, 180, 180)
	p.SetLineWidth(0.3)
	p.Line(15, y, 195, y)
	p.Ln(3)
}

func drawSummaryBox(p *fpdf.Fpdf, report *model.SalesReport) {
	p.SetFillColor(245, 245, 245)
	p.SetDrawColor(200, 200, 200)
	p.SetLineWidth(0.4)

	x := p.GetX()
	y := p.GetY()
	w := 180.0
	h := 32.0

	p.Rect(x, y, w, h, "FD")

	p.SetFont("Helvetica", "B", 11)
	p.SetXY(x+4, y+4)
	p.CellFormat(86, 6, "Total Omzet", "", 0, "L", false, 0, "")
	p.CellFormat(86, 6, "Jumlah Pesanan", "", 1, "L", false, 0, "")

	p.SetFont("Helvetica", "", 12)
	p.SetXY(x+4, y+11)
	p.CellFormat(86, 6, FormatRupiah(report.TotalRevenue), "", 0, "L", false, 0, "")
	p.CellFormat(86, 6, fmt.Sprintf("%d", report.TotalOrders), "", 1, "L", false, 0, "")

	p.SetFont("Helvetica", "B", 11)
	p.SetXY(x+4, y+20)
	p.CellFormat(86, 6, "Rata-rata Rating", "", 0, "L", false, 0, "")
	p.CellFormat(86, 6, "Jumlah Review", "", 1, "L", false, 0, "")

	p.SetFont("Helvetica", "", 12)
	p.SetXY(x+4, y+27)
	p.CellFormat(86, 6, fmt.Sprintf("%.1f / 5", report.AvgRating), "", 0, "L", false, 0, "")
	p.CellFormat(86, 6, fmt.Sprintf("%d", report.ReviewCount), "", 1, "L", false, 0, "")

	p.SetXY(x, y+h)
}

func drawStatusTable(p *fpdf.Fpdf, rows []model.StatusCount) {
	headers := []string{"Status", "Jumlah"}
	widths := []float64{90, 90}

	drawTableHeader(p, headers, widths)

	p.SetFont("Helvetica", "", 10)
	for _, row := range rows {
		drawTableRow(p, []string{
			statusLabel(row.Status),
			fmt.Sprintf("%d", row.Count),
		}, widths)
	}
}

func drawTopMenuTable(p *fpdf.Fpdf, rows []model.TopMenu) {
	if len(rows) == 0 {
		p.SetFont("Helvetica", "I", 10)
		p.CellFormat(0, 8, "Belum ada penjualan pada periode ini.", "", 1, "L", false, 0, "")
		return
	}

	headers := []string{"#", "Menu", "Terjual", "Omzet"}
	widths := []float64{10, 90, 30, 50}

	drawTableHeader(p, headers, widths)

	p.SetFont("Helvetica", "", 10)
	for _, row := range rows {
		drawTableRow(p, []string{
			fmt.Sprintf("%d", row.Rank),
			row.MenuName,
			fmt.Sprintf("%d", row.QtySold),
			FormatRupiah(row.Revenue),
		}, widths)
	}
}

func drawTableHeader(p *fpdf.Fpdf, headers []string, widths []float64) {
	p.SetFont("Helvetica", "B", 10)
	p.SetFillColor(60, 60, 60)
	p.SetTextColor(255, 255, 255)

	for i, header := range headers {
		align := "L"
		if i > 0 {
			align = "R"
		}
		if headers[0] == "#" && i == 0 {
			align = "C"
		}
		p.CellFormat(widths[i], 8, header, "1", 0, align, true, 0, "")
	}
	p.Ln(-1)

	p.SetTextColor(30, 30, 30)
}

func drawTableRow(p *fpdf.Fpdf, cells []string, widths []float64) {
	p.SetFont("Helvetica", "", 10)

	maxLines := 1
	lineWidths := make([][]string, len(cells))

	for i, cell := range cells {
		lines := p.SplitLines([]byte(cell), widths[i]-2)
		converted := make([]string, 0, len(lines))
		for _, line := range lines {
			converted = append(converted, string(line))
		}
		lineWidths[i] = converted
		if len(converted) > maxLines {
			maxLines = len(converted)
		}
	}

	height := float64(maxLines)*6 + 2

	xStart := p.GetX()
	yStart := p.GetY()

	for i := range cells {
		x := xStart
		for j := 0; j < i; j++ {
			x += widths[j]
		}

		p.SetXY(x, yStart)
		p.CellFormat(widths[i], height, "", "1", 0, "L", false, 0, "")

		p.SetXY(x+1, yStart+1)
		align := "L"
		if i > 0 {
			align = "R"
		}
		if cells[0] != "" && i == 0 && len(cells) >= 4 {
			align = "C"
		}
		for _, line := range lineWidths[i] {
			p.CellFormat(widths[i]-2, 6, line, "", 0, align, false, 0, "")
			p.Ln(-1)
			p.SetXY(x+1, p.GetY())
		}
	}

	p.SetXY(xStart, yStart+height)
}

func statusLabel(status string) string {
	switch status {
	case "menunggu":
		return "Menunggu"
	case "proses":
		return "Diproses"
	case "siap":
		return "Siap"
	case "selesai":
		return "Selesai"
	case "dibatalkan":
		return "Dibatalkan"
	default:
		return status
	}
}

// FormatRupiah memformat integer Rupiah dengan pemisah ribuan
// gaya Indonesia, contoh: 1234567 -> "Rp 1.234.567".
func FormatRupiah(amount int) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}

	digits := fmt.Sprintf("%d", amount)

	var out []byte
	for i, c := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, byte(c))
	}

	if negative {
		return "Rp -" + string(out)
	}
	return "Rp " + string(out)
}

// NowJakarta mengembalikan waktu sekarang yang diformat untuk
// dicetak di laporan.
func NowJakarta() string {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.Local
	}
	return time.Now().In(loc).Format("2006-01-02 15:04 WIB")
}
