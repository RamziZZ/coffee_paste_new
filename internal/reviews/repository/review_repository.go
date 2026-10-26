package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"coffee_paste_backend/internal/reviews/model"
)

// Kode error PostgreSQL.
const (
	pgForeignKeyViolation = "23503"
	pgUniqueViolation     = "23505"
)

type ReviewRepository struct {
	db *pgxpool.Pool
}

func NewReviewRepository(db *pgxpool.Pool) *ReviewRepository {
	return &ReviewRepository{db: db}
}

const reviewSelect = `
	SELECT
		r.id::text,
		r.order_id::text,
		o.order_code,
		r.user_id::text,
		u.name,
		r.rating,
		r.comment,
		r.created_at::text,
		r.updated_at::text
	FROM reviews r
	JOIN orders o ON o.id = r.order_id
	JOIN users u ON u.id = r.user_id
`

func scanReview(row pgx.Row) (*model.Review, error) {
	var r model.Review

	err := row.Scan(
		&r.ID,
		&r.OrderID,
		&r.OrderCode,
		&r.UserID,
		&r.CustomerName,
		&r.Rating,
		&r.Comment,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// LoadOrderContext mengambil status dan pemilik pesanan untuk validasi
// pembuatan review (status harus selesai, pesanan milik pelanggan tsb).
func (r *ReviewRepository) LoadOrderContext(
	ctx context.Context,
	orderID string,
) (*model.OrderContext, error) {

	var oc model.OrderContext

	err := r.db.QueryRow(
		ctx,
		`SELECT o.status::text, c.name, u.id::text
		 FROM orders o
		 LEFT JOIN customers c ON c.id = o.customer_id
		 LEFT JOIN users u ON u.id = c.user_id
		 WHERE o.id = $1`,
		orderID,
	).Scan(&oc.Status, &oc.CustomerName, &oc.CustomerUserID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrOrderNotReviewable
	}
	if err != nil {
		return nil, err
	}

	return &oc, nil
}

// Create menyimpan review baru. Order_id unik: review kedua untuk
// pesanan yang sama menghasilkan ErrAlreadyReviewed.
func (r *ReviewRepository) Create(
	ctx context.Context,
	in model.NewReview,
) (*model.Review, error) {

	var id string

	err := r.db.QueryRow(
		ctx,
		`INSERT INTO reviews (order_id, user_id, rating, comment)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id::text`,
		in.OrderID,
		in.UserID,
		in.Rating,
		in.Comment,
	).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgUniqueViolation:
				return nil, model.ErrAlreadyReviewed
			case pgForeignKeyViolation:
				return nil, model.ErrOrderNotReviewable
			}
		}
		return nil, err
	}

	return r.GetByID(ctx, id)
}

// GetByID mengambil satu review.
func (r *ReviewRepository) GetByID(
	ctx context.Context,
	id string,
) (*model.Review, error) {

	review, err := scanReview(
		r.db.QueryRow(ctx, reviewSelect+` WHERE r.id = $1`, id),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrReviewNotFound
	}
	if err != nil {
		return nil, err
	}

	return review, nil
}

// List mengambil daftar review (terbaru dulu) beserta jumlah totalnya.
func (r *ReviewRepository) List(
	ctx context.Context,
	f model.ListFilter,
) ([]model.Review, int, error) {

	conds := make([]string, 0)
	args := make([]any, 0)

	add := func(cond string, value any) {
		args = append(args, value)
		placeholder := fmt.Sprintf("$%d", len(args))
		conds = append(conds, strings.Replace(cond, "?", placeholder, 1))
	}

	if f.UserID != nil {
		add("r.user_id = ?", *f.UserID)
	}
	if f.MinRating != nil {
		add("r.rating >= ?", *f.MinRating)
	}
	if f.MaxRating != nil {
		add("r.rating <= ?", *f.MaxRating)
	}

	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	var total int

	err := r.db.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM reviews r
		 JOIN orders o ON o.id = r.order_id
		 JOIN users u ON u.id = r.user_id`+where,
		args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	listArgs := append(args, f.Limit, f.Offset)
	query := reviewSelect + where + fmt.Sprintf(
		" ORDER BY r.created_at DESC LIMIT $%d OFFSET $%d",
		len(args)+1,
		len(args)+2,
	)

	rows, err := r.db.Query(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	reviews := make([]model.Review, 0)

	for rows.Next() {
		review, err := scanReview(rows)
		if err != nil {
			return nil, 0, err
		}
		reviews = append(reviews, *review)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

// Delete menghapus review (moderasi admin).
func (r *ReviewRepository) Delete(
	ctx context.Context,
	id string,
) error {

	tag, err := r.db.Exec(ctx, `DELETE FROM reviews WHERE id = $1`, id)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return model.ErrReviewNotFound
	}

	return nil
}
