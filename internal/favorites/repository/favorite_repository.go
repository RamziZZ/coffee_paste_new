package repository

import (
	"context"
	"errors"

	"coffee_paste_backend/internal/favorites/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrCustomerNotFound = errors.New("customer tidak ditemukan")

type FavoriteRepository struct {
	db *pgxpool.Pool
}

func NewFavoriteRepository(db *pgxpool.Pool) *FavoriteRepository {
	return &FavoriteRepository{
		db: db,
	}
}

const favoriteSelect = `
	SELECT
		fm.menu_id::text,
		m.name,
		cat.name,
		m.price,
		m.image_url,
		fm.created_at
	FROM favorite_menus fm
	JOIN menus m ON m.id = fm.menu_id
	JOIN categories cat ON cat.id = m.category_id
`

// customerID mengubah users.id (dari JWT) menjadi customers.id.
func (r *FavoriteRepository) customerID(
	ctx context.Context,
	userID string,
) (string, error) {

	var customerID string

	err := r.db.QueryRow(
		ctx,
		`SELECT id::text FROM customers WHERE user_id = $1`,
		userID,
	).Scan(&customerID)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrCustomerNotFound
	}

	return customerID, err
}

// GetByUserID mengambil semua favorite milik customer.
func (r *FavoriteRepository) GetByUserID(
	ctx context.Context,
	userID string,
) ([]model.Favorite, error) {

	customerID, err := r.customerID(ctx, userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(
		ctx,
		favoriteSelect+`WHERE fm.customer_id = $1 ORDER BY fm.created_at DESC`,
		customerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favorites []model.Favorite

	for rows.Next() {
		var favorite model.Favorite

		err := rows.Scan(
			&favorite.MenuID,
			&favorite.Name,
			&favorite.Category,
			&favorite.Price,
			&favorite.ImageURL,
			&favorite.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		favorites = append(favorites, favorite)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return favorites, nil
}

// Create menambahkan menu ke favorite customer.
func (r *FavoriteRepository) Create(
	ctx context.Context,
	userID string,
	menuID string,
) (*model.Favorite, error) {

	customerID, err := r.customerID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var favorite model.Favorite

	_, err = r.db.Exec(
		ctx,
		`INSERT INTO favorite_menus (customer_id, menu_id)
		 VALUES ($1, $2)`,
		customerID,
		menuID,
	)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(
		ctx,
		favoriteSelect+`WHERE fm.customer_id = $1 AND fm.menu_id = $2`,
		customerID,
		menuID,
	).Scan(
		&favorite.MenuID,
		&favorite.Name,
		&favorite.Category,
		&favorite.Price,
		&favorite.ImageURL,
		&favorite.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &favorite, nil
}

// Delete menghapus menu dari favorite customer.
func (r *FavoriteRepository) Delete(
	ctx context.Context,
	userID string,
	menuID string,
) error {

	customerID, err := r.customerID(ctx, userID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(
		ctx,
		`DELETE FROM favorite_menus
		 WHERE customer_id = $1 AND menu_id = $2`,
		customerID,
		menuID,
	)

	return err
}
