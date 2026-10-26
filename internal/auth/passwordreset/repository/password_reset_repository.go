package repository

import (
	"context"
	"errors"
	"time"

	"coffee_paste_backend/internal/auth/passwordreset/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPasswordResetNotFound = errors.New("password reset token tidak ditemukan")

type PasswordResetRepository struct {
	db *pgxpool.Pool
}

func NewPasswordResetRepository(db *pgxpool.Pool) *PasswordResetRepository {
	return &PasswordResetRepository{
		db: db,
	}
}

func (r *PasswordResetRepository) Create(
	ctx context.Context,
	userID string,
	tokenHash string,
	expiresAt time.Time,
) error {

	query := `
		INSERT INTO password_resets (
			id, user_id, token_hash, expires_at
		)
		VALUES (gen_random_uuid(), $1, $2, $3)
	`

	_, err := r.db.Exec(
		ctx,
		query,
		userID,
		tokenHash,
		expiresAt,
	)

	return err
}

func (r *PasswordResetRepository) FindValidByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*model.PasswordReset, error) {

	query := `
		SELECT
			id::text,
			user_id::text,
			token_hash,
			expires_at,
			used_at
		FROM password_resets
		WHERE token_hash = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()
		ORDER BY expires_at DESC
		LIMIT 1
	`

	var reset model.PasswordReset

	err := r.db.QueryRow(
		ctx,
		query,
		tokenHash,
	).Scan(
		&reset.ID,
		&reset.UserID,
		&reset.TokenHash,
		&reset.ExpiresAt,
		&reset.UsedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPasswordResetNotFound
		}

		return nil, err
	}

	return &reset, nil
}

func (r *PasswordResetRepository) MarkAsUsed(
	ctx context.Context,
	id string,
) error {

	query := `
		UPDATE password_resets
		SET used_at = NOW()
		WHERE id = $1
		  AND used_at IS NULL
	`

	_, err := r.db.Exec(
		ctx,
		query,
		id,
	)

	return err
}
