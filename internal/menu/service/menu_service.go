package service

import (
	"context"
	"errors"
	"strings"

	"coffee_paste_backend/internal/menu/dto"
	"coffee_paste_backend/internal/menu/model"
	"coffee_paste_backend/internal/menu/repository"

	"github.com/google/uuid"
)

type MenuService struct {
	repository *repository.MenuRepository
}

func NewMenuService(
	repo *repository.MenuRepository,
) *MenuService {
	return &MenuService{repository: repo}
}

func (s *MenuService) GetAll(
	ctx context.Context,
	search string,
	categoryID string,
	status string,
) ([]*model.MenuItem, error) {
	categoryID = strings.TrimSpace(categoryID)
	if categoryID != "" {
		if _, err := uuid.Parse(categoryID); err != nil {
			return nil, errors.New("category_id tidak valid")
		}
	}

	return s.repository.GetAll(
		ctx,
		strings.TrimSpace(search),
		categoryID,
		strings.TrimSpace(status),
	)
}

func (s *MenuService) GetByID(
	ctx context.Context,
	id string,
) (*model.MenuItem, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("ID menu tidak valid")
	}

	return s.repository.GetByID(ctx, id)
}

func validateMenu(
	categoryID string,
	name string,
	price int,
	status string,
) error {
	if _, err := uuid.Parse(categoryID); err != nil {
		return errors.New("kategori menu harus dipilih")
	}
	if strings.TrimSpace(name) == "" {
		return errors.New("nama menu wajib diisi")
	}
	if price < 0 {
		return errors.New("harga menu tidak boleh negatif")
	}
	if status != "tersedia" && status != "habis" {
		return errors.New("status menu harus berupa tersedia atau habis")
	}

	return nil
}

func (s *MenuService) Create(
	ctx context.Context,
	req dto.CreateMenuRequest,
) (*model.MenuItem, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Status = strings.TrimSpace(req.Status)

	if err := validateMenu(
		req.CategoryID,
		req.Name,
		req.Price,
		req.Status,
	); err != nil {
		return nil, err
	}

	return s.repository.Create(ctx, req)
}

func (s *MenuService) Update(
	ctx context.Context,
	id string,
	req dto.UpdateMenuRequest,
) (*model.MenuItem, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("ID menu tidak valid")
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Status = strings.TrimSpace(req.Status)

	if err := validateMenu(
		req.CategoryID,
		req.Name,
		req.Price,
		req.Status,
	); err != nil {
		return nil, err
	}

	return s.repository.Update(ctx, id, req)
}

func (s *MenuService) Delete(
	ctx context.Context,
	id string,
) error {
	if _, err := uuid.Parse(id); err != nil {
		return errors.New("ID menu tidak valid")
	}

	return s.repository.Delete(ctx, id)
}

func (s *MenuService) GetFeaturedMenu(
	ctx context.Context,
) ([]*model.MenuItem, error) {
	return s.repository.GetFeaturedMenu(ctx)
}

func (s *MenuService) GetCategories(
	ctx context.Context,
) ([]*model.MenuCategory, error) {
	return s.repository.GetCategories(ctx)
}

func (s *MenuService) GetFilters(
	ctx context.Context,
) ([]*dto.MenuFilterResponse, error) {
	return s.repository.GetFilters(ctx)
}
