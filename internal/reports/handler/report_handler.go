package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"coffee_paste_backend/internal/reports/dto"
	"coffee_paste_backend/internal/reports/model"
	"coffee_paste_backend/internal/reports/service"
)

type ReportHandler struct {
	service *service.ReportService
}

func NewReportHandler(service *service.ReportService) *ReportHandler {
	return &ReportHandler{
		service: service,
	}
}

// respondError memetakan error service ke status HTTP yang sesuai.
// Detail error internal hanya dicatat di log, tidak dikirim ke client.
func respondError(c *gin.Context, err error, fallbackMessage string) {
	var validationErr *model.ValidationError

	switch {
	case errors.As(err, &validationErr):
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": validationErr.Message,
		})
	default:
		log.Println(fallbackMessage+":", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": fallbackMessage,
		})
	}
}

func bindQuery(c *gin.Context) (dto.SalesReportQuery, bool) {
	var query dto.SalesReportQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Data request tidak valid",
			"detail":  err.Error(),
		})
		return query, false
	}
	return query, true
}

// GetSalesReport mengambil laporan penjualan (JSON).
// GET /api/v1/reports/sales?period=daily|weekly|monthly&date=YYYY-MM-DD
func (h *ReportHandler) GetSalesReport(c *gin.Context) {
	query, ok := bindQuery(c)
	if !ok {
		return
	}

	report, err := h.service.GetSalesReport(c.Request.Context(), query)
	if err != nil {
		respondError(c, err, "Gagal membuat laporan penjualan")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   report,
	})
}

// ExportSalesPDF mengekspor laporan penjualan sebagai file PDF.
// GET /api/v1/reports/sales/pdf?period=daily|weekly|monthly&date=YYYY-MM-DD
func (h *ReportHandler) ExportSalesPDF(c *gin.Context) {
	query, ok := bindQuery(c)
	if !ok {
		return
	}

	content, filename, err := h.service.BuildSalesReportPDF(
		c.Request.Context(), query,
	)
	if err != nil {
		respondError(c, err, "Gagal membuat PDF laporan penjualan")
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header(
		"Content-Disposition",
		`attachment; filename="`+filename+`"`,
	)
	c.Header("Content-Length", strconv.Itoa(len(content)))
	c.Data(http.StatusOK, "application/pdf", content)
}
