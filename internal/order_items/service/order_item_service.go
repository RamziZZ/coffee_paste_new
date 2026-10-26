package service

import (
	"context"
	"errors"
	"strings"

	"coffee_paste_backend/internal/order_items/dto"
	"coffee_paste_backend/internal/order_items/model"
	"coffee_paste_backend/internal/order_items/repository"

	"github.com/google/uuid"
)

var ErrInvalidInput = errors.New("data input tidak valid")

type OrderItemService struct {
	repo *repository.OrderItemRepository
}

func NewOrderItemService(repo *repository.OrderItemRepository) *OrderItemService {
	return &OrderItemService{repo: repo}
}

func (s *OrderItemService) GetByOrderID(
	ctx context.Context,
	orderID uuid.UUID,
) ([]model.OrderItem, error) {
	return s.repo.GetByOrderID(ctx, orderID)
}

func (s *OrderItemService) Create(
	ctx context.Context,
	orderID uuid.UUID,
	req dto.CreateOrderItemRequest,
) (*model.OrderItem, error) {
	if req.Qty < 1 {
		return nil, ErrInvalidInput
	}

	menuID, err := uuid.Parse(req.MenuID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	if req.ItemNote != nil {
		note := strings.TrimSpace(*req.ItemNote)
		req.ItemNote = &note
	}

	item := &model.OrderItem{
		ID:       uuid.New(),
		OrderID:  orderID,
		MenuID:   menuID,
		Qty:      req.Qty,
		ItemNote: req.ItemNote,
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *OrderItemService) Update(
	ctx context.Context,
	orderID uuid.UUID,
	itemID uuid.UUID,
	req dto.UpdateOrderItemRequest,
) (*model.OrderItem, error) {
	if req.Qty < 1 {
		return nil, ErrInvalidInput
	}

	if req.ItemNote != nil {
		note := strings.TrimSpace(*req.ItemNote)
		req.ItemNote = &note
	}

	return s.repo.Update(ctx, orderID, itemID, req.Qty, req.ItemNote)
}

func (s *OrderItemService) Delete(
	ctx context.Context,
	orderID uuid.UUID,
	itemID uuid.UUID,
) error {
	return s.repo.Delete(ctx, orderID, itemID)
}
