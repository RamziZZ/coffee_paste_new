package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"coffee_paste_backend/internal/menu/dto"
	"coffee_paste_backend/internal/menu/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("menu tidak ditemukan")
	ErrInUse    = errors.New("menu tidak dapat dihapus karena masih digunakan pesanan atau stok")
)

type MenuRepository struct {
	db *pgxpool.Pool
}

func NewMenuRepository(db *pgxpool.Pool) *MenuRepository {
	return &MenuRepository{db: db}
}

const menuSelect = `
	SELECT
		id::text,
		category_id::text,
		name,
		description,
		price,
		status::text,
		image_url,
		is_featured
	FROM menus
`

func scanMenu(row pgx.Row) (*model.MenuItem, error) {
	var menu model.MenuItem

	err := row.Scan(
		&menu.ID,
		&menu.CategoryID,
		&menu.Name,
		&menu.Description,
		&menu.Price,
		&menu.Status,
		&menu.ImageURL,
		&menu.IsFeatured,
	)
	if err != nil {
		return nil, err
	}

	return &menu, nil
}

func (r *MenuRepository) GetAll(
	ctx context.Context,
	search string,
	categoryID string,
	status string,
) ([]*model.MenuItem, error) {
	query := menuSelect + `
		WHERE TRUE
	`
	args := make([]any, 0)
	argPos := 1

	if search != "" {
		query += fmt.Sprintf(
			" AND name ILIKE $%d",
			argPos,
		)
		args = append(args, "%"+search+"%")
		argPos++
	}

	if categoryID != "" {
		query += fmt.Sprintf(
			" AND category_id = $%d",
			argPos,
		)
		args = append(args, categoryID)
		argPos++
	}

	if status != "" {
		query += fmt.Sprintf(
			" AND status::text = $%d",
			argPos,
		)
		args = append(args, status)
		argPos++
	}

	query += " ORDER BY name ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	menus := make([]*model.MenuItem, 0)

	for rows.Next() {
		menu, err := scanMenu(rows)
		if err != nil {
			return nil, err
		}
		menus = append(menus, menu)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return menus, nil
}

func (r *MenuRepository) GetByID(
	ctx context.Context,
	id string,
) (*model.MenuItem, error) {
	menu, err := scanMenu(r.db.QueryRow(
		ctx,
		menuSelect+`WHERE id = $1`,
		id,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return menu, nil
}

func (r *MenuRepository) Create(
	ctx context.Context,
	req dto.CreateMenuRequest,
) (*model.MenuItem, error) {
	query := `
		INSERT INTO menus (
			id,
			category_id,
			name,
			description,
			price,
			status,
			image_url,
			is_featured
		)
		VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7
		)
		RETURNING
			id::text, category_id::text, name, description, price,
			status::text, image_url, is_featured
	`

	menu, err := scanMenu(r.db.QueryRow(
		ctx,
		query,
		req.CategoryID,
		strings.TrimSpace(req.Name),
		req.Description,
		req.Price,
		req.Status,
		req.ImageURL,
		req.IsFeatured,
	))
	if err != nil {
		return nil, err
	}

	return menu, nil
}

func (r *MenuRepository) Update(
	ctx context.Context,
	id string,
	req dto.UpdateMenuRequest,
) (*model.MenuItem, error) {
	query := `
		UPDATE menus
		SET
			category_id = $2,
			name = $3,
			description = $4,
			price = $5,
			status = $6,
			image_url = $7,
			is_featured = $8
		WHERE id = $1
		RETURNING
			id::text, category_id::text, name, description, price,
			status::text, image_url, is_featured
	`

	menu, err := scanMenu(r.db.QueryRow(
		ctx,
		query,
		id,
		req.CategoryID,
		strings.TrimSpace(req.Name),
		req.Description,
		req.Price,
		req.Status,
		req.ImageURL,
		req.IsFeatured,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return menu, nil
}

func (r *MenuRepository) Delete(
	ctx context.Context,
	id string,
) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM menus WHERE id = $1`, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return ErrInUse
		}
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *MenuRepository) GetFeaturedMenu(
	ctx context.Context,
) ([]*model.MenuItem, error) {
	rows, err := r.db.Query(ctx, menuSelect+`
		WHERE is_featured
		  AND status = 'tersedia'
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	menus := make([]*model.MenuItem, 0)

	for rows.Next() {
		menu, err := scanMenu(rows)
		if err != nil {
			return nil, err
		}
		menus = append(menus, menu)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return menus, nil
}

func (r *MenuRepository) GetCategories(
	ctx context.Context,
) ([]*model.MenuCategory, error) {
	query := `
		SELECT
			id::text,
			name,
			sort_order
		FROM categories
		ORDER BY sort_order ASC, name ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]*model.MenuCategory, 0)

	for rows.Next() {
		var category model.MenuCategory

		if err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.SortOrder,
		); err != nil {
			return nil, err
		}

		categories = append(categories, &category)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

// GetFilters memakai categories sebagai sumber filter menu
// (skema baru tidak punya filter_label/icon terpisah).
func (r *MenuRepository) GetFilters(
	ctx context.Context,
) ([]*dto.MenuFilterResponse, error) {
	query := `
		SELECT
			id::text,
			name,
			name AS filter_label,
			'' AS icon
		FROM categories
		ORDER BY sort_order ASC, name ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	filters := make([]*dto.MenuFilterResponse, 0)

	for rows.Next() {
		var filter dto.MenuFilterResponse

		if err := rows.Scan(
			&filter.ID,
			&filter.Name,
			&filter.FilterLabel,
			&filter.Icon,
		); err != nil {
			return nil, err
		}

		filters = append(filters, &filter)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return filters, nil
}
