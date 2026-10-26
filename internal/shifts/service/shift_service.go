package service

import (
	"context"
	"errors"
	"strings"

	"coffee_paste_backend/internal/shifts/dto"
	"coffee_paste_backend/internal/shifts/model"
	"coffee_paste_backend/internal/shifts/repository"
)

var ErrInvalidShiftData = errors.New("data shift tidak valid")

type ShiftService struct {
	repository *repository.ShiftRepository
}

func NewShiftService(
	repo *repository.ShiftRepository,
) *ShiftService {
	return &ShiftService{
		repository: repo,
	}
}

func (s *ShiftService) Open(
	ctx context.Context,
	userID string,
	req *dto.OpenShiftRequest,
) (*model.Shift, error) {
	if strings.TrimSpace(userID) == "" || req == nil || req.OpeningCash < 0 {
		return nil, ErrInvalidShiftData
	}

	if req.Notes != nil {
		notes := strings.TrimSpace(*req.Notes)
		req.Notes = &notes
	}

	return s.repository.Open(ctx, userID, req)
}

func (s *ShiftService) GetActive(
	ctx context.Context,
	userID string,
) (*model.Shift, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrInvalidShiftData
	}

	return s.repository.FindActiveByUserID(ctx, userID)
}

func (s *ShiftService) ListMine(
	ctx context.Context,
	userID string,
) ([]model.Shift, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrInvalidShiftData
	}

	return s.repository.ListByUserID(ctx, userID)
}

func (s *ShiftService) Close(
	ctx context.Context,
	id string,
	userID string,
	req *dto.CloseShiftRequest,
) (*model.Shift, error) {
	if strings.TrimSpace(id) == "" ||
		strings.TrimSpace(userID) == "" ||
		req == nil ||
		req.ClosingCash < 0 {
		return nil, ErrInvalidShiftData
	}

	if req.Notes != nil {
		notes := strings.TrimSpace(*req.Notes)
		req.Notes = &notes
	}

	return s.repository.Close(ctx, id, userID, req)
}
