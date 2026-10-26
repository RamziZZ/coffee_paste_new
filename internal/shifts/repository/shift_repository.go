package repository

import (
	"context"
	"errors"

	"coffee_paste_backend/internal/shifts/dto"
	"coffee_paste_backend/internal/shifts/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrShiftNotFound      = errors.New("shift tidak ditemukan")
	ErrShiftAlreadyOpen   = errors.New("kasir masih memiliki shift aktif")
	ErrShiftAlreadyClosed = errors.New("shift sudah ditutup")
)

type ShiftRepository struct {
	db *pgxpool.Pool
}

func NewShiftRepository(db *pgxpool.Pool) *ShiftRepository {
	return &ShiftRepository{db: db}
}

const shiftColumns = `
	id::text,
	user_id::text,
	opening_cash,
	closing_cash,
	expected_cash,
	cash_difference,
	status,
	opened_at,
	closed_at,
	notes,
	created_at,
	updated_at
`

func scanShift(row pgx.Row) (*model.Shift, error) {
	var shift model.Shift

	err := row.Scan(
		&shift.ID,
		&shift.UserID,
		&shift.OpeningCash,
		&shift.ClosingCash,
		&shift.ExpectedCash,
		&shift.CashDifference,
		&shift.Status,
		&shift.OpenedAt,
		&shift.ClosedAt,
		&shift.Notes,
		&shift.CreatedAt,
		&shift.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrShiftNotFound
		}
		return nil, err
	}

	return &shift, nil
}

func (r *ShiftRepository) Open(
	ctx context.Context,
	userID string,
	req *dto.OpenShiftRequest,
) (*model.Shift, error) {
	var shiftID string

	err := r.db.QueryRow(ctx, `
		INSERT INTO shifts (user_id, opening_cash, notes)
		SELECT $1, $2, $3
		WHERE NOT EXISTS (
			SELECT 1
			FROM shifts
			WHERE user_id = $1
			  AND status = 'open'
		)
		RETURNING id::text
	`,
		userID,
		req.OpeningCash,
		req.Notes,
	).Scan(&shiftID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrShiftAlreadyOpen
		}
		return nil, err
	}

	return r.FindByID(ctx, shiftID)
}

func (r *ShiftRepository) FindByID(
	ctx context.Context,
	id string,
) (*model.Shift, error) {
	row := r.db.QueryRow(ctx, `
		SELECT `+shiftColumns+`
		FROM shifts
		WHERE id = $1
		LIMIT 1
	`, id)

	return scanShift(row)
}

func (r *ShiftRepository) FindActiveByUserID(
	ctx context.Context,
	userID string,
) (*model.Shift, error) {
	row := r.db.QueryRow(ctx, `
		SELECT `+shiftColumns+`
		FROM shifts
		WHERE user_id = $1
		  AND status = 'open'
		ORDER BY opened_at DESC
		LIMIT 1
	`, userID)

	return scanShift(row)
}

func (r *ShiftRepository) ListByUserID(
	ctx context.Context,
	userID string,
) ([]model.Shift, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+shiftColumns+`
		FROM shifts
		WHERE user_id = $1
		ORDER BY opened_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shifts := make([]model.Shift, 0)

	for rows.Next() {
		shift, err := scanShift(rows)
		if err != nil {
			return nil, err
		}
		shifts = append(shifts, *shift)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return shifts, nil
}

func (r *ShiftRepository) Close(
	ctx context.Context,
	id string,
	userID string,
	req *dto.CloseShiftRequest,
) (*model.Shift, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE shifts
		SET
			closing_cash = $1,
			expected_cash = opening_cash,
			cash_difference = $1 - opening_cash,
			status = 'closed',
			closed_at = NOW(),
			notes = COALESCE($2, notes),
			updated_at = NOW()
		WHERE id = $3
		  AND user_id = $4
		  AND status = 'open'
		RETURNING `+shiftColumns,
		req.ClosingCash,
		req.Notes,
		id,
		userID,
	)

	shift, err := scanShift(row)
	if err != nil {
		return nil, err
	}

	return shift, nil
}
