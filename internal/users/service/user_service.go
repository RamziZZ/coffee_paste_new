package service

import (
	"context"
	"strings"

	"coffee_paste_backend/internal/users/dto"
	"coffee_paste_backend/internal/users/model"
	"coffee_paste_backend/internal/users/repository"
)

type UserService struct {
	repository *repository.UserRepository
}

func NewUserService(
	repo *repository.UserRepository,
) *UserService {
	return &UserService{repository: repo}
}

func (s *UserService) GetByID(
	ctx context.Context,
	id string,
) (*model.User, error) {
	return s.repository.FindByID(ctx, id)
}

func (s *UserService) List(
	ctx context.Context,
) ([]model.User, error) {
	return s.repository.List(ctx)
}

func (s *UserService) UpdateMyProfile(
	ctx context.Context,
	id string,
	req *dto.UpdateMyProfileRequest,
) (*model.User, error) {
	req.Name = strings.TrimSpace(req.Name)

	if req.Name == "" {
		return nil, ErrInvalidUserData
	}

	if req.Phone != nil {
		phone := strings.TrimSpace(*req.Phone)
		req.Phone = &phone
	}

	if req.PhotoURL != nil {
		photoURL := strings.TrimSpace(*req.PhotoURL)
		req.PhotoURL = &photoURL
	}

	return s.repository.UpdateProfile(ctx, id, req)
}

func (s *UserService) UpdateStatus(
	ctx context.Context,
	id string,
	isActive bool,
) (*model.User, error) {
	return s.repository.UpdateStatus(ctx, id, isActive)
}

var ErrInvalidUserData = errInvalidUserData{}

type errInvalidUserData struct{}

func (errInvalidUserData) Error() string {
	return "data pengguna tidak valid"
}
