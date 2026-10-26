package service

import (
	"context"
	"fmt"
	"strings"

	"coffee_paste_backend/internal/inventory/dto"
	"coffee_paste_backend/internal/inventory/model"
	"coffee_paste_backend/internal/inventory/repository"
)

const (
	defaultLimit = 20
	maxLimit     = 100

	// Tipe notifikasi stok menipis (harus sama dengan enum di database).
	notificationTypeStockLow = "stock_low"
)

// StockNotifier adalah port ke modul notifications. Boleh nil
// (notifikasi dilewati) supaya modul ini bisa dipakai mandiri.
type StockNotifier interface {
	NotifyStaff(
		ctx context.Context,
		notifType string,
		refID string,
		title string,
		body string,
	)
}

type InventoryService struct {
	repository *repository.InventoryRepository
	notifier   StockNotifier
}

func NewInventoryService(
	repository *repository.InventoryRepository,
	notifier StockNotifier,
) *InventoryService {
	return &InventoryService{
		repository: repository,
		notifier:   notifier,
	}
}

func toStockResponse(s model.StockRecord) dto.StockResponse {
	return dto.StockResponse{
		MenuID:            s.MenuID,
		MenuName:          s.MenuName,
		Qty:               s.Qty,
		LowStockThreshold: s.LowStockThreshold,
		IsLowStock:        s.Qty <= s.LowStockThreshold,
	}
}

// List mengambil daftar stok. low_stock=true hanya menampilkan menu
// yang qty-nya di bawah threshold.
func (s *InventoryService) List(
	ctx context.Context,
	q dto.ListInventoryQuery,
) ([]dto.StockResponse, dto.Meta, error) {

	page := q.Page
	if page < 1 {
		page = 1
	}

	limit := q.Limit
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	stocks, total, err := s.repository.List(ctx, model.ListFilter{
		LowOnly: q.LowStock,
		Limit:   limit,
		Offset:  (page - 1) * limit,
	})
	if err != nil {
		return nil, dto.Meta{}, err
	}

	result := make([]dto.StockResponse, 0, len(stocks))
	for _, rec := range stocks {
		result = append(result, toStockResponse(rec))
	}

	return result, dto.Meta{Page: page, Limit: limit, Total: total}, nil
}

// Get mengambil stok satu menu.
func (s *InventoryService) Get(
	ctx context.Context,
	menuID string,
) (*dto.StockResponse, error) {

	record, err := s.repository.GetByMenuID(ctx, menuID)
	if err != nil {
		return nil, err
	}

	response := toStockResponse(*record)
	return &response, nil
}

// Adjust menyesuaikan stok: set_qty mengatur langsung, change
// menambah/mengurangi. Menu baru otomatis mulai dilacak stoknya.
func (s *InventoryService) Adjust(
	ctx context.Context,
	req dto.AdjustStockRequest,
) (*dto.StockResponse, error) {

	if (req.Change == nil) == (req.SetQty == nil) {
		return nil, &model.ValidationError{
			Message: "Isi salah satu dari change atau set_qty (tidak boleh keduanya)",
		}
	}

	var record *model.StockRecord
	var err error

	if req.SetQty != nil {
		record, err = s.repository.SetQty(
			ctx, req.MenuID, *req.SetQty, req.LowStockThreshold,
		)
	} else {
		if *req.Change == 0 {
			return nil, &model.ValidationError{
				Message: "Nilai change tidak boleh nol",
			}
		}
		record, err = s.repository.AdjustQty(
			ctx, req.MenuID, *req.Change, req.LowStockThreshold,
		)
	}
	if err != nil {
		return nil, err
	}

	if record.Qty <= record.LowStockThreshold {
		s.notifyLowStock(ctx, []model.LowStockItem{{
			MenuID:    record.MenuID,
			MenuName:  record.MenuName,
			Qty:       record.Qty,
			Threshold: record.LowStockThreshold,
		}})
	}

	response := toStockResponse(*record)
	return &response, nil
}

// CheckAvailable memvalidasi ketersediaan stok untuk item pesanan.
// required: menu_id -> jumlah total yang dipesan. Menu yang tidak
// dilacak stoknya dianggap unlimited dan dilewati.
func (s *InventoryService) CheckAvailable(
	ctx context.Context,
	required map[string]int,
) error {

	if len(required) == 0 {
		return nil
	}

	tracked, err := s.repository.CheckAvailability(ctx, required)
	if err != nil {
		return err
	}

	insufficient := make([]string, 0)

	for _, rec := range tracked {
		wanted := required[rec.MenuID]
		if rec.Qty < wanted {
			insufficient = append(insufficient, fmt.Sprintf(
				"%s (sisa %d, diminta %d)",
				rec.MenuName, rec.Qty, wanted,
			))
		}
	}

	if len(insufficient) > 0 {
		return &model.ValidationError{
			Message: "Stok tidak mencukupi untuk: " +
				strings.Join(insufficient, "; "),
		}
	}

	return nil
}

// DeductForOrder memotong stok semua item dalam pesanan. Dipanggil
// saat pesanan berstatus selesai. Kegagalan pemotongan mengembalikan
// error supaya pemanggil bisa memutuskan apakah gagal fatal atau tidak.
func (s *InventoryService) DeductForOrder(
	ctx context.Context,
	orderID string,
) error {

	lowStocks, err := s.repository.DeductForOrder(ctx, orderID)
	if err != nil {
		return err
	}

	s.notifyLowStock(ctx, lowStocks)

	return nil
}

// notifyLowStock mengirim notifikasi stok menipis ke staff (best-effort).
func (s *InventoryService) notifyLowStock(
	ctx context.Context,
	items []model.LowStockItem,
) {
	if s.notifier == nil {
		return
	}

	for _, item := range items {
		s.notifier.NotifyStaff(
			ctx,
			notificationTypeStockLow,
			item.MenuID,
			"Stok menipis",
			fmt.Sprintf(
				"%s tersisa %d (threshold %d)",
				item.MenuName, item.Qty, item.Threshold,
			),
		)
	}
}
