package service

import (
	"context"
	"errors"

	"coffee_paste_backend/internal/payments/dto"
	"coffee_paste_backend/internal/payments/model"
	"coffee_paste_backend/internal/payments/repository"

	"github.com/google/uuid"
)

type PaymentService struct {
	Repository *repository.PaymentRepository
}

func NewPaymentService(
	repo *repository.PaymentRepository,
) *PaymentService {
	return &PaymentService{Repository: repo}
}

// Create membuat pembayaran pending untuk pesanan.
// receivedBy adalah users.id kasir/admin yang memproses pembayaran.
func (s *PaymentService) Create(
	ctx context.Context,
	req dto.CreatePaymentRequest,
	receivedBy string,
) (*model.Payment, error) {
	if _, err := uuid.Parse(req.OrderID); err != nil {
		return nil, errors.New("order_id tidak valid")
	}
	if _, err := uuid.Parse(receivedBy); err != nil {
		return nil, errors.New("penerima pembayaran tidak valid")
	}
	if req.Amount <= 0 {
		return nil, errors.New("amount harus lebih dari 0")
	}
	if req.CashReceived != nil && *req.CashReceived < 0 {
		return nil, errors.New("cash_received tidak boleh negatif")
	}

	payment := &model.Payment{
		OrderID:      req.OrderID,
		ReceivedBy:   receivedBy,
		Method:       req.Method,
		Amount:       req.Amount,
		CashReceived: req.CashReceived,
		Status:       "pending",
		ReferenceNo:  req.ReferenceNo,
	}

	return s.Repository.Create(ctx, payment)
}

func (s *PaymentService) GetByID(
	ctx context.Context,
	id string,
) (*model.Payment, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("id pembayaran tidak valid")
	}

	return s.Repository.GetByID(ctx, id)
}

func (s *PaymentService) GetByOrderID(
	ctx context.Context,
	orderID string,
) ([]model.Payment, error) {
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, errors.New("order_id tidak valid")
	}

	return s.Repository.GetByOrderID(ctx, orderID)
}

func (s *PaymentService) UpdateStatus(
	ctx context.Context,
	id string,
	req dto.UpdatePaymentStatusRequest,
) (*model.Payment, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("id pembayaran tidak valid")
	}

	if req.CashReceived != nil && *req.CashReceived < 0 {
		return nil, errors.New("cash_received tidak boleh negatif")
	}

	return s.Repository.UpdateStatus(
		ctx,
		id,
		req.Status,
		req.CashReceived,
	)
}
