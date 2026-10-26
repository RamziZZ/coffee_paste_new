package repository

import (
	"context"
	"errors"

	"coffee_paste_backend/internal/auth/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user tidak ditemukan")

type AuthRepository struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{
		db: db,
	}
}

// userSelect adalah kolom users pada skema baru (uuid, is_active boolean,
// tanpa address/gender/birth_*/updated_at/deleted_at — data profil kasir
// disimpan di tabel terpisah kasir_profiles).
const userSelect = `
	SELECT
		id::text,
		role::text,
		name,
		email,
		password_hash,
		phone,
		photo_url,
		is_active,
		created_at,
		pending_email
	FROM users
`

func scanUser(row pgx.Row) (*model.User, error) {
	var user model.User

	err := row.Scan(
		&user.ID,
		&user.Role,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Phone,
		&user.PhotoURL,
		&user.IsActive,
		&user.CreatedAt,
		&user.PendingEmail,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *AuthRepository) FindByEmail(
	ctx context.Context,
	email string,
) (*model.User, error) {

	user, err := scanUser(r.db.QueryRow(
		ctx,
		userSelect+`WHERE LOWER(email) = LOWER($1) LIMIT 1`,
		email,
	))

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return user, nil
}

func (r *AuthRepository) FindByID(
	ctx context.Context,
	id string,
) (*model.User, error) {

	user, err := scanUser(r.db.QueryRow(
		ctx,
		userSelect+`WHERE id = $1 LIMIT 1`,
		id,
	))

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return user, nil
}

func (r *AuthRepository) UpdatePassword(
	ctx context.Context,
	userID string,
	passwordHash string,
) error {

	_, err := r.db.Exec(
		ctx,
		`UPDATE users SET password_hash = $1 WHERE id = $2`,
		passwordHash,
		userID,
	)

	return err
}

// CreateCustomer membuat user customer sekaligus baris customers
// (dibutuhkan orders dan favorites yang merujuk customers.id)
// dalam satu transaksi.
func (r *AuthRepository) CreateCustomer(
	ctx context.Context,
	name string,
	email string,
	passwordHash string,
	phone string,
) (*model.User, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID string

	err = tx.QueryRow(
		ctx,
		`INSERT INTO users (
			id, role, name, email, password_hash, phone, is_active
		)
		VALUES (
			gen_random_uuid(), 'customer', $1, $2, $3, NULLIF($4, ''), TRUE
		)
		RETURNING id::text`,
		name,
		email,
		passwordHash,
		phone,
	).Scan(&userID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO customers (id, user_id, name, email, phone)
		 VALUES (gen_random_uuid(), $1, $2, $3, NULLIF($4, ''))`,
		userID,
		name,
		email,
		phone,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.FindByID(ctx, userID)
}
