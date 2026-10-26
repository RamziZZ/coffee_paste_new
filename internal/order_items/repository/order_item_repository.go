package repository

import (
	"context"
	"errors"

	"coffee_paste_backend/internal/order_items/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("order item atau menu tidak ditemukan")

type OrderItemRepository struct {
	db *pgxpool.Pool
}

func NewOrderItemRepository(db *pgxpool.Pool) *OrderItemRepository {
	return &OrderItemRepository{db: db}
}

func (r *OrderItemRepository) GetByOrderID(
	ctx context.Context,
	orderID uuid.UUID,
) ([]model.OrderItem, error) {
	query := `
		SELECT id, order_id, menu_id, menu_name, price, qty, item_note
		FROM order_items
		WHERE order_id = $1
		ORDER BY menu_name ASC
	`

	rows, err := r.db.Query(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.OrderItem, 0)

	for rows.Next() {
		var item model.OrderItem

		err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.MenuID,
			&item.MenuName,
			&item.Price,
			&item.Qty,
			&item.ItemNote,
		)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *OrderItemRepository) Create(
	ctx context.Context,
	item *model.OrderItem,
) error {
	query := `
		INSERT INTO order_items (
			id, order_id, menu_id, menu_name, price, qty, item_note
		)
		SELECT $1, $2, m.id, m.name, m.price, $4, $5
		FROM menus m
		WHERE m.id = $3
		RETURNING menu_name, price
	`

	err := r.db.QueryRow(
		ctx,
		query,
		item.ID,
		item.OrderID,
		item.MenuID,
		item.Qty,
		item.ItemNote,
	).Scan(&item.MenuName, &item.Price)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	return err
}

func (r *OrderItemRepository) Update(
	ctx context.Context,
	orderID uuid.UUID,
	itemID uuid.UUID,
	qty int,
	itemNote *string,
) (*model.OrderItem, error) {
	query := `
		UPDATE order_items
		SET qty = $3, item_note = $4
		WHERE order_id = $1 AND id = $2
		RETURNING id, order_id, menu_id, menu_name, price, qty, item_note
	`

	var item model.OrderItem

	err := r.db.QueryRow(
		ctx,
		query,
		orderID,
		itemID,
		qty,
		itemNote,
	).Scan(
		&item.ID,
		&item.OrderID,
		&item.MenuID,
		&item.MenuName,
		&item.Price,
		&item.Qty,
		&item.ItemNote,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *OrderItemRepository) Delete(
	ctx context.Context,
	orderID uuid.UUID,
	itemID uuid.UUID,
) error {
	query := `
		DELETE FROM order_items
		WHERE order_id = $1 AND id = $2
	`

	result, err := r.db.Exec(ctx, query, orderID, itemID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
