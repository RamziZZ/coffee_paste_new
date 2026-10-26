package repository

import (
	"context"
	"errors"

	"coffee_paste_backend/internal/users/dto"
	"coffee_paste_backend/internal/users/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user tidak ditemukan")

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func scanUser(row pgx.Row) (*model.User, error) {
	var user model.User

	err := row.Scan(
		&user.ID,
		&user.Role,
		&user.Name,
		&user.Email,
		&user.Phone,
		&user.PhotoURL,
		&user.IsActive,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

const userColumns = `
	id::text,
	role::text,
	name,
	email,
	phone,
	photo_url,
	is_active,
	created_at
`

func (r *UserRepository) FindByID(
	ctx context.Context,
	id string,
) (*model.User, error) {
	query := `
		SELECT ` + userColumns + `
		FROM users
		WHERE id = $1
		LIMIT 1
	`

	return scanUser(r.db.QueryRow(ctx, query, id))
}

func (r *UserRepository) List(
	ctx context.Context,
) ([]model.User, error) {
	query := `
		SELECT ` + userColumns + `
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]model.User, 0)

	for rows.Next() {
		var user model.User

		err := rows.Scan(
			&user.ID,
			&user.Role,
			&user.Name,
			&user.Email,
			&user.Phone,
			&user.PhotoURL,
			&user.IsActive,
			&user.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) UpdateProfile(
	ctx context.Context,
	id string,
	req *dto.UpdateMyProfileRequest,
) (*model.User, error) {
	query := `
		UPDATE users
		SET
			name = $1,
			phone = $2,
			photo_url = $3
		WHERE id = $4
		RETURNING ` + userColumns

	return scanUser(r.db.QueryRow(
		ctx,
		query,
		req.Name,
		req.Phone,
		req.PhotoURL,
		id,
	))
}

func (r *UserRepository) UpdateStatus(
	ctx context.Context,
	id string,
	isActive bool,
) (*model.User, error) {
	query := `
		UPDATE users
		SET is_active = $1
		WHERE id = $2
		RETURNING ` + userColumns

	return scanUser(r.db.QueryRow(
		ctx,
		query,
		isActive,
		id,
	))
}
