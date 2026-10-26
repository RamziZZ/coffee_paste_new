package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"coffee_paste_backend/internal/inventory/model"
)

// Kode error PostgreSQL untuk pelanggaran foreign key.
const pgForeignKeyViolation = "23503"

type InventoryRepository struct {
	db *pgxpool.Pool
}

func NewInventoryRepository(db *pgxpool.Pool) *InventoryRepository {
	return &InventoryRepository{db: db}
}

const stockSelect = `
	SELECT
		s.menu_id::text,
		m.name,
		s.qty,
		s.low_stock_threshold,
		s.updated_at::text
	FROM inventory_stocks s
	JOIN menus m ON m.id = s.menu_id
`

func scanStock(row pgx.Row) (*model.StockRecord, error) {
	var s model.StockRecord

	err := row.Scan(
		&s.MenuID,
		&s.MenuName,
		&s.Qty,
		&s.LowStockThreshold,
		&s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// List mengambil daftar stok (yang menipis dulu kalau LowOnly,
// selain itu urut nama menu) beserta jumlah totalnya.
func (r *InventoryRepository) List(
	ctx context.Context,
	f model.ListFilter,
) ([]model.StockRecord, int, error) {

	where := ""
	if f.LowOnly {
		where = " WHERE s.qty <= s.low_stock_threshold"
	}

	var total int

	err := r.db.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM inventory_stocks s
		 JOIN menus m ON m.id = s.menu_id`+where,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	order := " ORDER BY m.name ASC"
	if f.LowOnly {
		order = " ORDER BY s.qty ASC, m.name ASC"
	}

	query := stockSelect + where + order + " LIMIT $1 OFFSET $2"

	rows, err := r.db.Query(ctx, query, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	stocks := make([]model.StockRecord, 0)

	for rows.Next() {
		s, err := scanStock(rows)
		if err != nil {
			return nil, 0, err
		}
		stocks = append(stocks, *s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return stocks, total, nil
}

// GetByMenuID mengambil satu baris stok berdasarkan menu_id.
func (r *InventoryRepository) GetByMenuID(
	ctx context.Context,
	menuID string,
) (*model.StockRecord, error) {

	s, err := scanStock(
		r.db.QueryRow(ctx, stockSelect+` WHERE s.menu_id = $1`, menuID),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrStockNotFound
	}
	if err != nil {
		return nil, err
	}

	return s, nil
}

// SetQty mengatur stok menu secara langsung (upsert). Menu yang belum
// punya baris stok akan dibuat otomatis (mulai melacak stok).
func (r *InventoryRepository) SetQty(
	ctx context.Context,
	menuID string,
	qty int,
	threshold *int,
) (*model.StockRecord, error) {

	var s model.StockRecord

	err := r.db.QueryRow(
		ctx,
		`INSERT INTO inventory_stocks (menu_id, qty, low_stock_threshold)
		 VALUES ($1, $2, COALESCE($3, 0))
		 ON CONFLICT (menu_id) DO UPDATE SET
			qty = EXCLUDED.qty,
			low_stock_threshold = COALESCE(
				$3, inventory_stocks.low_stock_threshold
			),
			updated_at = NOW()
		 RETURNING menu_id::text, qty, low_stock_threshold, updated_at::text`,
		menuID,
		qty,
		threshold,
	).Scan(&s.MenuID, &s.Qty, &s.LowStockThreshold, &s.UpdatedAt)
	if err != nil {
		return nil, r.mapMenuFK(err)
	}

	return r.hydrate(ctx, &s)
}

// AdjustQty menambah/mengurangi stok menu secara atomik. Baris stok
// yang belum ada dibuat dengan qty awal delta (harus >= 0). Mengembalikan
// ErrInsufficientStock kalau hasil penyesuaian akan negatif.
func (r *InventoryRepository) AdjustQty(
	ctx context.Context,
	menuID string,
	delta int,
	threshold *int,
) (*model.StockRecord, error) {

	var s model.StockRecord

	err := r.db.QueryRow(
		ctx,
		`UPDATE inventory_stocks
		 SET qty = qty + $2,
		     low_stock_threshold = COALESCE($3, low_stock_threshold),
		     updated_at = NOW()
		 WHERE menu_id = $1 AND qty + $2 >= 0
		 RETURNING menu_id::text, qty, low_stock_threshold`,
		menuID,
		delta,
		threshold,
	).Scan(&s.MenuID, &s.Qty, &s.LowStockThreshold)

	if errors.Is(err, pgx.ErrNoRows) {
		// Belum ada baris stok: buat baru dengan qty = delta.
		if delta < 0 {
			return nil, model.ErrInsufficientStock
		}

		err = r.db.QueryRow(
			ctx,
			`INSERT INTO inventory_stocks (menu_id, qty, low_stock_threshold)
			 VALUES ($1, $2, COALESCE($3, 0))
			 RETURNING menu_id::text, qty, low_stock_threshold`,
			menuID,
			delta,
			threshold,
		).Scan(&s.MenuID, &s.Qty, &s.LowStockThreshold)
		if err != nil {
			return nil, r.mapMenuFK(err)
		}
	} else if err != nil {
		return nil, err
	}

	return r.hydrate(ctx, &s)
}

// CheckAvailability mengambil stok terlacak untuk menu yang diminta.
// Menu yang tidak punya baris stok dianggap unlimited dan tidak
// dimasukkan ke hasil. required: menu_id -> jumlah total yang diminta.
func (r *InventoryRepository) CheckAvailability(
	ctx context.Context,
	required map[string]int,
) ([]model.StockRecord, error) {

	if len(required) == 0 {
		return nil, nil
	}

	ids := make([]string, 0, len(required))
	for menuID := range required {
		ids = append(ids, menuID)
	}

	rows, err := r.db.Query(
		ctx,
		stockSelect+` WHERE s.menu_id = ANY($1::uuid[])`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tracked := make([]model.StockRecord, 0, len(ids))

	for rows.Next() {
		s, err := scanStock(rows)
		if err != nil {
			return nil, err
		}
		tracked = append(tracked, *s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tracked, nil
}

// DeductForOrder memotong stok semua menu dalam satu pesanan berdasarkan
// qty order_items. Stok tidak pernah menjadi negatif (dibatasi nol).
// Mengembalikan menu yang setelah pemotongan berada di bawah threshold.
func (r *InventoryRepository) DeductForOrder(
	ctx context.Context,
	orderID string,
) ([]model.LowStockItem, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(
		ctx,
		`UPDATE inventory_stocks s
		 SET qty = GREATEST(s.qty - i.qty, 0),
		     updated_at = NOW()
		 FROM order_items i
		 WHERE i.order_id = $1
		   AND s.menu_id = i.menu_id`,
		orderID,
	)
	if err != nil {
		return nil, err
	}

	rows, err := tx.Query(
		ctx,
		`SELECT s.menu_id::text, m.name, s.qty, s.low_stock_threshold
		 FROM inventory_stocks s
		 JOIN menus m ON m.id = s.menu_id
		 WHERE s.menu_id IN (
		 	SELECT i.menu_id FROM order_items i WHERE i.order_id = $1
		 )
		   AND s.qty <= s.low_stock_threshold
		 ORDER BY s.qty ASC`,
		orderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lowStocks := make([]model.LowStockItem, 0)

	for rows.Next() {
		var item model.LowStockItem

		if err := rows.Scan(
			&item.MenuID,
			&item.MenuName,
			&item.Qty,
			&item.Threshold,
		); err != nil {
			return nil, err
		}
		lowStocks = append(lowStocks, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return lowStocks, nil
}

// hydrate mengisi nama menu (dan waktu update) untuk satu record stok.
func (r *InventoryRepository) hydrate(
	ctx context.Context,
	s *model.StockRecord,
) (*model.StockRecord, error) {

	record, err := scanStock(
		r.db.QueryRow(ctx, stockSelect+` WHERE s.menu_id = $1`, s.MenuID),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrStockNotFound
	}
	if err != nil {
		return nil, err
	}

	return record, nil
}

// mapMenuFK memetakan pelanggaran foreign key (menu_id) ke error domain.
func (r *InventoryRepository) mapMenuFK(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
		return model.ErrMenuNotFound
	}
	return err
}
