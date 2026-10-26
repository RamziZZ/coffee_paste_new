package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"coffee_paste_backend/config"
	"coffee_paste_backend/internal/payments/midtrans/dto"
	"coffee_paste_backend/internal/payments/midtrans/model"
	"coffee_paste_backend/internal/payments/midtrans/repository"
	"coffee_paste_backend/pkg/midtrans"

	"github.com/google/uuid"
)

var ErrPaymentNotFound = errors.New("pembayaran tidak ditemukan")

// PaymentNotifier adalah port ke modul notifications. Boleh nil
// (notifikasi dilewati). Semua metode best-effort, tidak mengembalikan error.
type PaymentNotifier interface {
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

// Tipe notifikasi pembayaran (harus sama dengan enum di database).
const notificationTypePaymentPaid = "payment_paid"

type MidtransService struct {
	Repository *repository.MidtransRepository
	Client     *midtrans.Client
	Config     config.Config
	notifier   PaymentNotifier
}

func NewMidtransService(
	repo *repository.MidtransRepository,
	cfg config.Config,
	notifier PaymentNotifier,
) *MidtransService {
	client := midtrans.NewClient(
		cfg.MidtransServerKey,
		cfg.MidtransMerchantName,
		cfg.MidtransEnvironment,
	)

	return &MidtransService{
		Repository: repo,
		Client:     client,
		Config:     cfg,
		notifier:   notifier,
	}
}

// CreateSnap membuat snap transaction untuk pesanan tertentu.
func (s *MidtransService) CreateSnap(
	ctx context.Context,
	req dto.CreateSnapPaymentRequest,
) (*dto.SnapPaymentResponse, error) {

	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		return nil, errors.New("order_id wajib diisi")
	}

	total, paid, _, err := s.Repository.GetOrderInfo(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if paid {
		return nil, errors.New("pesanan sudah dibayar")
	}

	if total <= 0 {
		return nil, errors.New("total pesanan harus lebih dari 0")
	}

	existing, err := s.Repository.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if existing != nil && existing.Status == "pending" {
		resp := &dto.SnapPaymentResponse{
			PaymentID: existing.ID,
			OrderID:   existing.OrderID,
			Amount:    existing.Amount,
			Status:    existing.Status,
		}
		if existing.SnapToken != nil {
			resp.SnapToken = *existing.SnapToken
		}
		if existing.RedirectURL != nil {
			resp.RedirectURL = *existing.RedirectURL
		}
		return resp, nil
	}

	items, err := s.Repository.GetOrderItems(ctx, orderID)
	if err != nil {
		return nil, err
	}

	midtransItems := make([]midtrans.ItemDetail, 0, len(items))
	for _, it := range items {
		midtransItems = append(midtransItems, midtrans.ItemDetail{
			ID:    it.MenuID,
			Price: it.Price,
			Qty:   it.Qty,
			Name:  it.MenuName,
		})
	}

	customerName, customerEmail, customerPhone, err := s.Repository.GetCustomerInfo(ctx, orderID)
	if err != nil {
		return nil, err
	}

	var customerDetail *midtrans.CustomerDetail
	if customerName != "" {
		customerDetail = &midtrans.CustomerDetail{
			FirstName: customerName,
		}
		if customerEmail != nil {
			customerDetail.Email = *customerEmail
		}
		if customerPhone != nil {
			customerDetail.Phone = *customerPhone
		}
	}

	snapReq := midtrans.SnapRequest{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:     orderID,
			GrossAmount: total,
		},
		ItemDetails:     midtransItems,
		CustomerDetails: customerDetail,
		Expiry: &midtrans.Expiry{
			Unit:     "minutes",
			Duration: 60,
		},
	}

	snapResp, err := s.Client.CreateSnapTransaction(snapReq)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat transaksi Midtrans: %w", err)
	}

	payment := &model.MidtransPayment{
		OrderID:     orderID,
		SnapToken:   &snapResp.Token,
		RedirectURL: &snapResp.RedirectURL,
		Amount:      total,
		Status:      "pending",
	}

	created, err := s.Repository.Create(ctx, payment)
	if err != nil {
		return nil, err
	}

	resp := &dto.SnapPaymentResponse{
		PaymentID: created.ID,
		OrderID:   created.OrderID,
		Amount:    created.Amount,
		Status:    created.Status,
	}
	if created.SnapToken != nil {
		resp.SnapToken = *created.SnapToken
	}
	if created.RedirectURL != nil {
		resp.RedirectURL = *created.RedirectURL
	}

	return resp, nil
}

// HandleNotification memproses notifikasi (webhook) dari Midtrans.
func (s *MidtransService) HandleNotification(
	ctx context.Context,
	notification midtrans.Notification,
) error {

	if !s.Client.VerifyNotificationSignature(notification) {
		return errors.New("signature notifikasi tidak valid")
	}

	localStatus := midtrans.StatusMapping(
		notification.TransactionStatus,
		notification.FraudStatus,
	)

	// Kolom status memakai enum payment_status (success), bukan 'paid'.
	dbStatus := localStatus
	if dbStatus == "paid" {
		dbStatus = "success"
	}

	var bank *string
	if notification.Bank != "" {
		bank = &notification.Bank
	}

	var maskedCard *string
	if notification.MaskedCard != "" {
		maskedCard = &notification.MaskedCard
	}

	_, err := s.Repository.UpdateFromNotification(
		ctx,
		notification.TransactionID,
		notification.OrderID,
		dbStatus,
		notification.PaymentType,
		bank,
		maskedCard,
		notification.FraudStatus,
	)
	if err != nil {
		return err
	}

	if localStatus == "paid" {
		s.notifyPaymentPaid(ctx, notification.OrderID)
	}

	return nil
}

// notifyPaymentPaid mengirim notifikasi pembayaran diterima ke
// customer pemilik pesanan dan ke staff (best-effort).
func (s *MidtransService) notifyPaymentPaid(
	ctx context.Context,
	orderID string,
) {
	if s.notifier == nil {
		return
	}

	customerUserID, orderCode, err := s.Repository.GetOrderNotifyInfo(
		ctx, orderID,
	)
	if err != nil {
		log.Printf(
			"Gagal mengambil info pesanan %s untuk notifikasi: %v",
			orderID, err,
		)
		return
	}

	if customerUserID != nil {
		s.notifier.Notify(
			ctx,
			*customerUserID,
			notificationTypePaymentPaid,
			orderID,
			"Pembayaran diterima",
			fmt.Sprintf(
				"Pembayaran untuk pesanan %s sudah kami terima.",
				orderCode,
			),
		)
	}

	s.notifier.NotifyStaff(
		ctx,
		notificationTypePaymentPaid,
		orderID,
		"Pembayaran diterima",
		fmt.Sprintf(
			"Pembayaran pesanan %s via Midtrans sudah diterima.",
			orderCode,
		),
	)
}

// GetByOrderID mengambil info pembayaran Midtrans untuk pesanan.
func (s *MidtransService) GetByOrderID(
	ctx context.Context,
	orderID string,
) (*model.MidtransPayment, error) {

	if strings.TrimSpace(orderID) == "" {
		return nil, errors.New("order_id wajib diisi")
	}

	payment, err := s.Repository.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if payment == nil {
		return nil, ErrPaymentNotFound
	}

	return payment, nil
}

// GetByID mengambil pembayaran Midtrans berdasarkan ID.
func (s *MidtransService) GetByID(
	ctx context.Context,
	id string,
) (*model.MidtransPayment, error) {

	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("id pembayaran tidak valid")
	}

	payment, err := s.Repository.GetByID(ctx, id)
	if err != nil {
		if err.Error() == "pembayaran tidak ditemukan" {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}

	return payment, nil
}

// GetClientKey mengambil Midtrans client key untuk frontend.
func (s *MidtransService) GetClientKey() string {
	return s.Config.MidtransClientKey
}

// GetEnvironment mengambil environment Midtrans (sandbox/production).
func (s *MidtransService) GetEnvironment() string {
	return s.Config.MidtransEnvironment
}
