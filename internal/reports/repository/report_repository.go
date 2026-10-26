package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"coffee_paste_backend/internal/reports/model"
)

type ReportRepository struct {
	db *pgxpool.Pool
}

func NewReportRepository(db *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{db: db}
}

// GetSalesSummary mengambil ringkasan penjualan (omzet, jumlah pesanan,
// dan jumlah per status) untuk rentang [start, end).
// Omzet = SUM(item) - discount untuk pesanan berstatus selesai.
func (r *ReportRepository) GetSalesSummary(
	ctx context.Context,
	start, end time.Time,
) (*model.SummaryRow, error) {

	var row model.SummaryRow

	err := r.db.QueryRow(ctx, `
		WITH order_totals AS (
			SELECT
				o.id,
				o.status::text AS status,
				o.discount,
				COALESCE(
					SUM(oi.price * oi.qty), 0
				)::int AS subtotal
			FROM orders o
			LEFT JOIN order_items oi ON oi.order_id = o.id
			WHERE o.ordered_at >= $1 AND o.ordered_at < $2
			GROUP BY o.id
		)
		SELECT
			COALESCE(
				SUM(subtotal - discount)
				FILTER (WHERE status = 'selesai'), 0
			)::int,
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE status = 'menunggu')::int,
			COUNT(*) FILTER (WHERE status = 'proses')::int,
			COUNT(*) FILTER (WHERE status = 'siap')::int,
			COUNT(*) FILTER (WHERE status = 'selesai')::int,
			COUNT(*) FILTER (WHERE status = 'dibatalkan')::int
		FROM order_totals
	`, start, end).Scan(
		&row.TotalRevenue,
		&row.TotalOrders,
		&row.Menunggu,
		&row.Proses,
		&row.Siap,
		&row.Selesai,
		&row.Dibatalkan,
	)
	if err != nil {
		return nil, err
	}

	return &row, nil
}

// GetTopMenus mengambil menu terlaris (berdasarkan qty terjual) untuk
// pesanan berstatus selesai dalam rentang [start, end).
func (r *ReportRepository) GetTopMenus(
	ctx context.Context,
	start, end time.Time,
	limit int,
) ([]model.TopMenu, error) {

	rows, err := r.db.Query(ctx, `
		SELECT
			oi.menu_id::text,
			oi.menu_name,
			COALESCE(SUM(oi.qty), 0)::int,
			COALESCE(SUM(oi.price * oi.qty), 0)::int
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE o.status = 'selesai'
		  AND o.ordered_at >= $1 AND o.ordered_at < $2
		GROUP BY oi.menu_id, oi.menu_name
		ORDER BY SUM(oi.qty) DESC, SUM(oi.price * oi.qty) DESC
		LIMIT $3
	`, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	menus := make([]model.TopMenu, 0)

	for rows.Next() {
		var m model.TopMenu

		if err := rows.Scan(
			&m.MenuID,
			&m.MenuName,
			&m.QtySold,
			&m.Revenue,
		); err != nil {
			return nil, err
		}

		m.Rank = len(menus) + 1
		menus = append(menus, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return menus, nil
}

// GetRatingSummary mengambil rata-rata rating dan jumlah review
// yang dibuat dalam rentang [start, end).
func (r *ReportRepository) GetRatingSummary(
	ctx context.Context,
	start, end time.Time,
) (*model.RatingRow, error) {

	var row model.RatingRow

	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(AVG(r.rating), 0)::float8,
			COUNT(*)::int
		FROM reviews r
		WHERE r.created_at >= $1 AND r.created_at < $2
	`, start, end).Scan(&row.AvgRating, &row.ReviewCount)
	if err != nil {
		return nil, err
	}

	return &row, nil
}
