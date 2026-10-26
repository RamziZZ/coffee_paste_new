package repository

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"coffee_paste_backend/internal/orders/model"
)

// Kode error PostgreSQL untuk pelanggaran foreign key.
const pgForeignKeyViolation = "23503"

// Huruf kode pesanan tanpa karakter yang mirip (0/O, 1/I).
const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// Query dasar pesanan. Subtotal dihitung dari order_items karena
// tabel orders tidak menyimpan subtotal maupun total.
const orderSelect = `
	SELECT
		o.id::text,
		o.order_code,
		o.customer_id::text,
		c.user_id::text,
		c.name,
		o.cashier_id::text,
		o.table_number,
		o.source::text,
		o.order_type::text,
		o.status::text,
		COALESCE((SELECT SUM(oi.price * oi.qty) FROM order_items oi WHERE oi.order_id = o.id), 0)::int,
		o.discount,
		o.note,
		o.cancel_reason,
		COALESCE((SELECT SUM(oi.qty) FROM order_items oi WHERE oi.order_id = o.id), 0)::int,
		o.ordered_at
	FROM orders o
	LEFT JOIN customers c ON c.id = o.customer_id
`

type OrderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{
		db: db,
	}
}

func scanOrder(row pgx.Row) (*model.Order, error) {
	var o model.Order

	err := row.Scan(
		&o.ID,
		&o.OrderCode,
		&o.CustomerID,
		&o.CustomerUserID,
		&o.CustomerName,
		&o.CashierID,
		&o.TableNumber,
		&o.Source,
		&o.OrderType,
		&o.Status,
		&o.Subtotal,
		&o.Discount,
		&o.Note,
		&o.CancelReason,
		&o.ItemCount,
		&o.OrderedAt,
	)
	if err != nil {
		return nil, err
	}

	return &o, nil
}

// generateOrderCode membuat kode seperti CP-261012-K7M2Q
// (tanggal yymmdd + 5 karakter acak).
func generateOrderCode() (string, error) {
	limit := big.NewInt(int64(len(codeAlphabet)))
	suffix := make([]byte, 5)

	for i := range suffix {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		suffix[i] = codeAlphabet[n.Int64()]
	}

	return "CP-" + time.Now().Format("060102") + "-" + string(suffix), nil
}

type menuInfo struct {
	Name   string
	Price  int
	Status string
}

// loadMenus mengambil nama, harga, dan status menu yang dipesan.
func loadMenus(
	ctx context.Context,
	tx pgx.Tx,
	items []model.NewItem,
) (map[string]menuInfo, error) {

	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.MenuID)
	}

	rows, err := tx.Query(
		ctx,
		`SELECT id::text, name, price, status::text
		 FROM menus
		 WHERE id::text = ANY($1)`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	menus := make(map[string]menuInfo)

	for rows.Next() {
		var id string
		var m menuInfo

		if err := rows.Scan(&id, &m.Name, &m.Price, &m.Status); err != nil {
			return nil, err
		}
		menus[id] = m
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return menus, nil
}

// GetCustomerID mencari customers.id berdasarkan users.id.
func (r *OrderRepository) GetCustomerID(
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
		return "", model.ErrCustomerNotFound
	}
	if err != nil {
		return "", err
	}

	return customerID, nil
}

// Create membuat pesanan beserta item dan log statusnya dalam satu
// transaksi. Nama dan harga menu disalin dari tabel menus (snapshot),
// bukan dari input client. Mengembalikan id pesanan baru.
func (r *OrderRepository) Create(
	ctx context.Context,
	in model.NewOrder,
) (string, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	menus, err := loadMenus(ctx, tx, in.Items)
	if err != nil {
		return "", err
	}

	subtotal := 0

	for _, it := range in.Items {
		menu, ok := menus[it.MenuID]
		if !ok {
			return "", &model.ValidationError{Message: "Ada menu yang tidak ditemukan"}
		}
		if menu.Status != model.MenuStatusTersedia {
			return "", &model.ValidationError{
				Message: fmt.Sprintf("Menu %q sedang tidak tersedia", menu.Name),
			}
		}
		subtotal += menu.Price * it.Qty
	}

	if in.Discount > subtotal {
		return "", &model.ValidationError{Message: "Diskon melebihi total pesanan"}
	}

	code, err := generateOrderCode()
	if err != nil {
		return "", err
	}

	var orderID string

	err = tx.QueryRow(
		ctx,
		`INSERT INTO orders (
			id, order_code, customer_id, cashier_id, table_number,
			source, order_type, status, discount, note
		)
		VALUES (
			gen_random_uuid(), $1, $2, $3, $4,
			$5::text::order_source, $6::text::order_type, $7::text::order_status, $8, $9
		)
		RETURNING id::text`,
		code,
		in.CustomerID,
		in.CashierID,
		in.TableNumber,
		in.Source,
		in.OrderType,
		model.StatusMenunggu,
		in.Discount,
		in.Note,
	).Scan(&orderID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
			return "", model.ErrCustomerNotFound
		}
		return "", err
	}

	for _, it := range in.Items {
		menu := menus[it.MenuID]

		_, err := tx.Exec(
			ctx,
			`INSERT INTO order_items (
				id, order_id, menu_id, menu_name, price, qty, item_note
			)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)`,
			orderID,
			it.MenuID,
			menu.Name,
			menu.Price,
			it.Qty,
			it.ItemNote,
		)
		if err != nil {
			return "", err
		}
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO order_status_logs (id, order_id, changed_by, status, note)
		 VALUES (gen_random_uuid(), $1, $2, $3::text::order_status, $4)`,
		orderID,
		in.ChangedBy,
		model.StatusMenunggu,
		"Pesanan dibuat",
	)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return orderID, nil
}

// GetByID mengambil satu pesanan.
func (r *OrderRepository) GetByID(
	ctx context.Context,
	orderID string,
) (*model.Order, error) {

	o, err := scanOrder(
		r.db.QueryRow(ctx, orderSelect+` WHERE o.id = $1`, orderID),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}

	return o, nil
}

// GetItems mengambil semua item dalam satu pesanan.
func (r *OrderRepository) GetItems(
	ctx context.Context,
	orderID string,
) ([]model.OrderItem, error) {

	rows, err := r.db.Query(
		ctx,
		`SELECT id::text, menu_id::text, menu_name, price, qty, item_note
		 FROM order_items
		 WHERE order_id = $1
		 ORDER BY menu_name`,
		orderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.OrderItem, 0)

	for rows.Next() {
		var it model.OrderItem

		if err := rows.Scan(
			&it.ID,
			&it.MenuID,
			&it.MenuName,
			&it.Price,
			&it.Qty,
			&it.ItemNote,
		); err != nil {
			return nil, err
		}
		items = append(items, it)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

// GetStatusLogs mengambil riwayat perubahan status, dari yang paling lama.
func (r *OrderRepository) GetStatusLogs(
	ctx context.Context,
	orderID string,
) ([]model.StatusLog, error) {

	rows, err := r.db.Query(
		ctx,
		`SELECT l.status::text, l.note, l.changed_by::text, u.name, l.changed_at
		 FROM order_status_logs l
		 JOIN users u ON u.id = l.changed_by
		 WHERE l.order_id = $1
		 ORDER BY l.changed_at ASC`,
		orderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]model.StatusLog, 0)

	for rows.Next() {
		var l model.StatusLog

		if err := rows.Scan(
			&l.Status,
			&l.Note,
			&l.ChangedBy,
			&l.ChangedByName,
			&l.ChangedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}

// List mengambil daftar pesanan (terbaru dulu) beserta jumlah totalnya.
func (r *OrderRepository) List(
	ctx context.Context,
	f model.ListFilter,
) ([]model.Order, int, error) {

	conds := make([]string, 0)
	args := make([]any, 0)

	add := func(cond string, value any) {
		args = append(args, value)
		placeholder := fmt.Sprintf("$%d", len(args))
		conds = append(conds, strings.Replace(cond, "?", placeholder, 1))
	}

	if f.CustomerUserID != nil {
		add("c.user_id = ?", *f.CustomerUserID)
	}
	if f.Status != nil {
		add("o.status = ?::text::order_status", *f.Status)
	}
	if f.DateFrom != nil {
		add("o.ordered_at >= ?::text::date", *f.DateFrom)
	}
	if f.DateTo != nil {
		add("o.ordered_at < (?::text::date + 1)", *f.DateTo)
	}

	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	var total int

	err := r.db.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM orders o
		 LEFT JOIN customers c ON c.id = o.customer_id`+where,
		args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	listArgs := append(args, f.Limit, f.Offset)
	query := orderSelect + where + fmt.Sprintf(
		" ORDER BY o.ordered_at DESC LIMIT $%d OFFSET $%d",
		len(args)+1,
		len(args)+2,
	)

	rows, err := r.db.Query(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	orders := make([]model.Order, 0)

	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, *o)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// UpdateStatus memindahkan status pesanan dan mencatat log-nya dalam
// satu transaksi. Baris pesanan dikunci (FOR UPDATE) supaya dua
// perubahan bersamaan tidak saling menimpa.
func (r *OrderRepository) UpdateStatus(
	ctx context.Context,
	change model.StatusChange,
) error {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current string

	err = tx.QueryRow(
		ctx,
		`SELECT status::text FROM orders WHERE id = $1 FOR UPDATE`,
		change.OrderID,
	).Scan(&current)

	if errors.Is(err, pgx.ErrNoRows) {
		return model.ErrOrderNotFound
	}
	if err != nil {
		return err
	}

	if change.RequireCurrent != "" && current != change.RequireCurrent {
		return model.ErrInvalidTransition
	}
	if !model.CanTransition(current, change.NewStatus) {
		return model.ErrInvalidTransition
	}

	_, err = tx.Exec(
		ctx,
		`UPDATE orders
		 SET status = $2::text::order_status,
		     cancel_reason = COALESCE($3, cancel_reason)
		 WHERE id = $1`,
		change.OrderID,
		change.NewStatus,
		change.CancelReason,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO order_status_logs (id, order_id, changed_by, status, note)
		 VALUES (gen_random_uuid(), $1, $2, $3::text::order_status, $4)`,
		change.OrderID,
		change.ChangedBy,
		change.NewStatus,
		change.Note,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
