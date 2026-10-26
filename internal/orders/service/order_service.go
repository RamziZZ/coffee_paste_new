package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"coffee_paste_backend/internal/orders/dto"
	"coffee_paste_backend/internal/orders/model"
	"coffee_paste_backend/internal/orders/repository"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// Actor adalah user yang sedang login (dari JWT).
type Actor struct {
	UserID string
	Role   string
}

func (a Actor) isStaff() bool {
	return a.Role == model.RoleKasir || a.Role == model.RoleAdmin
}

// StockChecker adalah port ke modul inventory. Boleh nil (pengecekan
// dan pemotongan stok dilewati) supaya orders bisa dipakai mandiri.
type StockChecker interface {
	// CheckAvailable memvalidasi stok. required: menu_id -> jumlah total.
	CheckAvailable(ctx context.Context, required map[string]int) error
	// DeductForOrder memotong stok item dalam pesanan (dipanggil saat selesai).
	DeductForOrder(ctx context.Context, orderID string) error
}

// OrderNotifier adalah port ke modul notifications. Boleh nil
// (notifikasi dilewati) supaya orders bisa dipakai mandiri.
// Semua metode bersifat best-effort dan tidak boleh mengembalikan error.
type OrderNotifier interface {
	Notify(
		ctx context.Context,
		userID string,
		notifType string,
		refID string,
		title string,
		body string,
	)
	NotifyStaff(
		ctx context.Context,
		notifType string,
		refID string,
		title string,
		body string,
	)
}

// Tipe notifikasi pesanan (harus sama dengan enum di database).
const (
	notificationTypeOrderCreated = "order_created"
	notificationTypeOrderStatus  = "order_status"
)

type OrderService struct {
	repository *repository.OrderRepository
	stock      StockChecker
	notifier   OrderNotifier
}

func NewOrderService(
	repository *repository.OrderRepository,
	stock StockChecker,
	notifier OrderNotifier,
) *OrderService {
	return &OrderService{
		repository: repository,
		stock:      stock,
		notifier:   notifier,
	}
}

// cleanPtr merapikan string opsional: kosong menjadi nil.
func cleanPtr(s *string) *string {
	if s == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func validStatus(s string) bool {
	switch s {
	case model.StatusMenunggu,
		model.StatusProses,
		model.StatusSiap,
		model.StatusSelesai,
		model.StatusDibatalkan:
		return true
	}
	return false
}

func parseDate(value string) (*string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	if _, err := time.Parse("2006-01-02", value); err != nil {
		return nil, &model.ValidationError{
			Message: "Format tanggal harus YYYY-MM-DD",
		}
	}

	return &value, nil
}

// authorizeView: staf boleh melihat semua pesanan, customer hanya
// pesanannya sendiri. Pesanan orang lain dianggap tidak ada.
func authorizeView(actor Actor, o *model.Order) error {
	if actor.isStaff() {
		return nil
	}

	if actor.Role == model.RoleCustomer &&
		o.CustomerUserID != nil &&
		strings.EqualFold(*o.CustomerUserID, actor.UserID) {
		return nil
	}

	return model.ErrOrderNotFound
}

func toOrderResponse(o model.Order) dto.OrderResponse {
	return dto.OrderResponse{
		ID:           o.ID,
		OrderCode:    o.OrderCode,
		CustomerID:   o.CustomerID,
		CustomerName: o.CustomerName,
		CashierID:    o.CashierID,
		TableNumber:  o.TableNumber,
		Source:       o.Source,
		OrderType:    o.OrderType,
		Status:       o.Status,
		Subtotal:     o.Subtotal,
		Discount:     o.Discount,
		Total:        o.Total(),
		Note:         o.Note,
		CancelReason: o.CancelReason,
		ItemCount:    o.ItemCount,
		OrderedAt:    o.OrderedAt,
	}
}

// detail mengambil pesanan lengkap dengan item dan riwayat statusnya.
func (s *OrderService) detail(
	ctx context.Context,
	orderID string,
) (*dto.OrderResponse, error) {

	order, err := s.repository.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	items, err := s.repository.GetItems(ctx, orderID)
	if err != nil {
		return nil, err
	}

	logs, err := s.repository.GetStatusLogs(ctx, orderID)
	if err != nil {
		return nil, err
	}

	response := toOrderResponse(*order)

	response.Items = make([]dto.OrderItemResponse, 0, len(items))
	for _, it := range items {
		response.Items = append(response.Items, dto.OrderItemResponse{
			ID:       it.ID,
			MenuID:   it.MenuID,
			MenuName: it.MenuName,
			Price:    it.Price,
			Qty:      it.Qty,
			ItemNote: it.ItemNote,
			Subtotal: it.Price * it.Qty,
		})
	}

	response.StatusLogs = make([]dto.StatusLogResponse, 0, len(logs))
	for _, l := range logs {
		response.StatusLogs = append(response.StatusLogs, dto.StatusLogResponse{
			Status:        l.Status,
			Note:          l.Note,
			ChangedBy:     l.ChangedBy,
			ChangedByName: l.ChangedByName,
			ChangedAt:     l.ChangedAt,
		})
	}

	return &response, nil
}

// Create membuat pesanan baru.
//   - customer: sumber customer_app, pesanan otomatis atas namanya, tanpa diskon.
//   - kasir/admin: sumber kasir, tercatat sebagai kasir, boleh mengisi
//     customer_id (opsional) dan diskon.
func (s *OrderService) Create(
	ctx context.Context,
	actor Actor,
	req dto.CreateOrderRequest,
) (*dto.OrderResponse, error) {

	items := make([]model.NewItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, model.NewItem{
			MenuID:   strings.ToLower(it.MenuID),
			Qty:      it.Qty,
			ItemNote: cleanPtr(it.ItemNote),
		})
	}

	table := cleanPtr(req.TableNumber)

	switch req.OrderType {
	case model.TypeDineIn:
		if table == nil {
			return nil, &model.ValidationError{
				Message: "Nomor meja wajib diisi untuk pesanan dine in",
			}
		}
	default:
		table = nil
	}

	in := model.NewOrder{
		TableNumber: table,
		OrderType:   req.OrderType,
		Note:        cleanPtr(req.Note),
		ChangedBy:   actor.UserID,
		Items:       items,
	}

	switch actor.Role {
	case model.RoleCustomer:
		customerID, err := s.repository.GetCustomerID(ctx, actor.UserID)
		if err != nil {
			return nil, err
		}

		in.CustomerID = &customerID
		in.Source = model.SourceCustomerApp
		in.Discount = 0

	case model.RoleKasir, model.RoleAdmin:
		cashierID := actor.UserID

		in.CashierID = &cashierID
		in.CustomerID = cleanPtr(req.CustomerID)
		in.Source = model.SourceKasir
		in.Discount = req.Discount

	default:
		return nil, model.ErrForbidden
	}

	if s.stock != nil {
		required := make(map[string]int)
		for _, it := range items {
			required[it.MenuID] += it.Qty
		}
		if err := s.stock.CheckAvailable(ctx, required); err != nil {
			return nil, err
		}
	}

	orderID, err := s.repository.Create(ctx, in)
	if err != nil {
		return nil, err
	}

	s.notifyOrderCreated(ctx, orderID)

	return s.detail(ctx, orderID)
}

// notifyOrderCreated mengirim notifikasi pesanan baru (best-effort):
// ke customer (kalau tercatat) dan ke staff (kalau dari aplikasi customer).
func (s *OrderService) notifyOrderCreated(
	ctx context.Context,
	orderID string,
) {
	if s.notifier == nil {
		return
	}

	order, err := s.repository.GetByID(ctx, orderID)
	if err != nil {
		log.Printf(
			"Gagal mengambil pesanan %s untuk notifikasi: %v",
			orderID, err,
		)
		return
	}

	title := "Pesanan diterima"
	body := fmt.Sprintf("Pesanan %s sudah kami terima.", order.OrderCode)

	if order.CustomerUserID != nil {
		s.notifier.Notify(
			ctx,
			*order.CustomerUserID,
			notificationTypeOrderCreated,
			orderID,
			title,
			body,
		)
	}

	if order.Source == model.SourceCustomerApp {
		s.notifier.NotifyStaff(
			ctx,
			notificationTypeOrderCreated,
			orderID,
			"Pesanan baru",
			fmt.Sprintf(
				"Pesanan %s dari aplikasi pelanggan.",
				order.OrderCode,
			),
		)
	}
}

// notifyOrderStatusChange mengirim notifikasi perubahan status ke
// customer pemilik pesanan (best-effort).
func (s *OrderService) notifyOrderStatusChange(
	ctx context.Context,
	order *model.Order,
	status string,
) {
	if s.notifier == nil || order.CustomerUserID == nil {
		return
	}

	var title string
	switch status {
	case model.StatusProses:
		title = "Pesanan sedang diproses"
	case model.StatusSiap:
		title = "Pesanan siap diambil"
	case model.StatusSelesai:
		title = "Pesanan selesai"
	case model.StatusDibatalkan:
		title = "Pesanan dibatalkan"
	default:
		return
	}

	s.notifier.Notify(
		ctx,
		*order.CustomerUserID,
		notificationTypeOrderStatus,
		order.ID,
		title,
		fmt.Sprintf("Pesanan %s: %s.", order.OrderCode, title),
	)
}

// Get mengambil detail satu pesanan.
func (s *OrderService) Get(
	ctx context.Context,
	actor Actor,
	orderID string,
) (*dto.OrderResponse, error) {

	order, err := s.repository.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if err := authorizeView(actor, order); err != nil {
		return nil, err
	}

	return s.detail(ctx, orderID)
}

// List mengambil daftar pesanan. Customer hanya melihat pesanannya sendiri.
func (s *OrderService) List(
	ctx context.Context,
	actor Actor,
	q dto.ListOrdersQuery,
) ([]dto.OrderResponse, dto.Meta, error) {

	filter := model.ListFilter{}

	switch {
	case actor.isStaff():
		// staf melihat semua pesanan
	case actor.Role == model.RoleCustomer:
		userID := actor.UserID
		filter.CustomerUserID = &userID
	default:
		return nil, dto.Meta{}, model.ErrForbidden
	}

	if status := strings.TrimSpace(q.Status); status != "" {
		if !validStatus(status) {
			return nil, dto.Meta{}, &model.ValidationError{
				Message: "Status tidak valid",
			}
		}
		filter.Status = &status
	}

	dateFrom, err := parseDate(q.DateFrom)
	if err != nil {
		return nil, dto.Meta{}, err
	}
	filter.DateFrom = dateFrom

	dateTo, err := parseDate(q.DateTo)
	if err != nil {
		return nil, dto.Meta{}, err
	}
	filter.DateTo = dateTo

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

	filter.Limit = limit
	filter.Offset = (page - 1) * limit

	orders, total, err := s.repository.List(ctx, filter)
	if err != nil {
		return nil, dto.Meta{}, err
	}

	result := make([]dto.OrderResponse, 0, len(orders))
	for _, o := range orders {
		result = append(result, toOrderResponse(o))
	}

	return result, dto.Meta{Page: page, Limit: limit, Total: total}, nil
}

// UpdateStatus mengubah status pesanan (khusus kasir dan admin).
func (s *OrderService) UpdateStatus(
	ctx context.Context,
	actor Actor,
	orderID string,
	req dto.UpdateStatusRequest,
) (*dto.OrderResponse, error) {

	if !actor.isStaff() {
		return nil, model.ErrForbidden
	}

	note := cleanPtr(req.Note)

	var cancelReason *string

	if req.Status == model.StatusDibatalkan {
		cancelReason = cleanPtr(req.CancelReason)
		if cancelReason == nil {
			return nil, &model.ValidationError{
				Message: "Alasan pembatalan wajib diisi",
			}
		}
		if note == nil {
			note = cancelReason
		}
	}

	err := s.repository.UpdateStatus(ctx, model.StatusChange{
		OrderID:      orderID,
		NewStatus:    req.Status,
		Note:         note,
		CancelReason: cancelReason,
		ChangedBy:    actor.UserID,
	})
	if err != nil {
		return nil, err
	}

	if order, err := s.repository.GetByID(ctx, orderID); err == nil {
		s.notifyOrderStatusChange(ctx, order, req.Status)
	} else {
		log.Printf(
			"Gagal mengambil pesanan %s untuk notifikasi: %v",
			orderID, err,
		)
	}

	// Pesanan selesai: potong stok menu (best-effort, tidak menggagalkan).
	if req.Status == model.StatusSelesai && s.stock != nil {
		if err := s.stock.DeductForOrder(ctx, orderID); err != nil {
			log.Printf(
				"Gagal memotong stok untuk pesanan %s: %v",
				orderID, err,
			)
		}
	}

	return s.detail(ctx, orderID)
}

// Cancel membatalkan pesanan oleh customer. Hanya pesanan miliknya
// yang masih berstatus menunggu yang boleh dibatalkan.
func (s *OrderService) Cancel(
	ctx context.Context,
	actor Actor,
	orderID string,
	req dto.CancelOrderRequest,
) (*dto.OrderResponse, error) {

	if actor.Role != model.RoleCustomer {
		return nil, model.ErrForbidden
	}

	order, err := s.repository.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if err := authorizeView(actor, order); err != nil {
		return nil, err
	}

	reason := cleanPtr(req.Reason)
	if reason == nil {
		defaultReason := "Dibatalkan oleh pelanggan"
		reason = &defaultReason
	}

	err = s.repository.UpdateStatus(ctx, model.StatusChange{
		OrderID:        orderID,
		NewStatus:      model.StatusDibatalkan,
		Note:           reason,
		CancelReason:   reason,
		ChangedBy:      actor.UserID,
		RequireCurrent: model.StatusMenunggu,
	})
	if err != nil {
		return nil, err
	}

	s.notifyOrderStatusChange(ctx, order, model.StatusDibatalkan)

	return s.detail(ctx, orderID)
}
