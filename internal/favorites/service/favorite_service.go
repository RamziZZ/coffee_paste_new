package service

import (
	"context"
	"errors"

	"coffee_paste_backend/internal/favorites/model"
	"coffee_paste_backend/internal/favorites/repository"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotCustomer   = errors.New("akun customer tidak ditemukan")
	ErrMenuNotFound  = errors.New("menu tidak ditemukan")
	ErrAlreadyExists = errors.New("menu sudah ada di favorite")
)

type FavoriteService struct {
	repository *repository.FavoriteRepository
}

func NewFavoriteService(
	repository *repository.FavoriteRepository,
) *FavoriteService {
	return &FavoriteService{
		repository: repository,
	}
}

// mapError menerjemahkan error database ke error domain.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, repository.ErrCustomerNotFound) {
		return ErrNotCustomer
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			// Pelanggaran FK: menu_id tidak ada di menus.
			return ErrMenuNotFound
		case "23505":
			// Pelanggaran unique: sudah pernah difavoritkan.
			return ErrAlreadyExists
		}
	}

	return err
}

// GetByUserID mengambil daftar favorite milik customer.
func (s *FavoriteService) GetByUserID(
	ctx context.Context,
	userID string,
) ([]model.Favorite, error) {

	favorites, err := s.repository.GetByUserID(ctx, userID)
	return favorites, mapError(err)
}

// Create menambahkan menu ke favorite customer.
func (s *FavoriteService) Create(
	ctx context.Context,
	userID string,
	menuID string,
) (*model.Favorite, error) {

	favorite, err := s.repository.Create(ctx, userID, menuID)
	return favorite, mapError(err)
}

// Delete menghapus menu dari favorite customer.
func (s *FavoriteService) Delete(
	ctx context.Context,
	userID string,
	menuID string,
) error {

	return mapError(s.repository.Delete(ctx, userID, menuID))
}
