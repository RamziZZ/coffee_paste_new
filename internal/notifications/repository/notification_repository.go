package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"coffee_paste_backend/internal/notifications/model"
)

type NotificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

const notificationSelect = `
	SELECT
		id::text,
		user_id::text,
		title,
		body,
		type,
		ref_id,
		is_read,
		created_at::text
	FROM notifications
`

func scanNotification(row pgx.Row) (*model.Notification, error) {
	var n model.Notification

	err := row.Scan(
		&n.ID,
		&n.UserID,
		&n.Title,
		&n.Body,
		&n.Type,
		&n.RefID,
		&n.IsRead,
		&n.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &n, nil
}

// Create menyimpan satu notifikasi.
func (r *NotificationRepository) Create(
	ctx context.Context,
	in model.NewNotification,
) error {

	_, err := r.db.Exec(
		ctx,
		`INSERT INTO notifications (user_id, title, body, type, ref_id)
		 VALUES ($1, $2, $3, $4, $5)`,
		in.UserID,
		in.Title,
		in.Body,
		in.Type,
		in.RefID,
	)
	return err
}

// CreateBatch menyimpan banyak notifikasi dalam satu transaksi
// (dipakai untuk broadcast ke staff).
func (r *NotificationRepository) CreateBatch(
	ctx context.Context,
	notifications []model.NewNotification,
) error {

	if len(notifications) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	batch := &pgx.Batch{}

	for _, n := range notifications {
		batch.Queue(
			`INSERT INTO notifications (user_id, title, body, type, ref_id)
			 VALUES ($1, $2, $3, $4, $5)`,
			n.UserID,
			n.Title,
			n.Body,
			n.Type,
			n.RefID,
		)
	}

	br := tx.SendBatch(ctx, batch)

	for range notifications {
		if _, err := br.Exec(); err != nil {
			_ = br.Close()
			return err
		}
	}

	if err := br.Close(); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// List mengambil notifikasi milik satu user (terbaru dulu)
// beserta jumlah totalnya.
func (r *NotificationRepository) List(
	ctx context.Context,
	f model.ListFilter,
) ([]model.Notification, int, error) {

	where := " WHERE user_id = $1"

	var total int

	err := r.db.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM notifications`+where,
		f.UserID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := notificationSelect + where +
		fmt.Sprintf(" ORDER BY created_at DESC LIMIT $2 OFFSET $3")

	rows, err := r.db.Query(ctx, query, f.UserID, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	notifications := make([]model.Notification, 0)

	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, 0, err
		}
		notifications = append(notifications, *n)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

// CountUnread menghitung notifikasi yang belum dibaca milik satu user.
func (r *NotificationRepository) CountUnread(
	ctx context.Context,
	userID string,
) (int, error) {

	var count int

	err := r.db.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM notifications
		 WHERE user_id = $1 AND is_read = FALSE`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// MarkRead menandai satu notifikasi milik user sebagai sudah dibaca.
func (r *NotificationRepository) MarkRead(
	ctx context.Context,
	userID string,
	notificationID string,
) error {

	tag, err := r.db.Exec(
		ctx,
		`UPDATE notifications
		 SET is_read = TRUE
		 WHERE id = $1 AND user_id = $2 AND is_read = FALSE`,
		notificationID,
		userID,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		// Sudah dibaca atau bukan miliknya: cek apakah notifikasi ada.
		var exists bool
		err := r.db.QueryRow(
			ctx,
			`SELECT EXISTS(
				SELECT 1 FROM notifications WHERE id = $1 AND user_id = $2
			)`,
			notificationID,
			userID,
		).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return model.ErrNotificationNotFound
		}
	}

	return nil
}

// MarkAllRead menandai semua notifikasi milik user sebagai sudah dibaca.
func (r *NotificationRepository) MarkAllRead(
	ctx context.Context,
	userID string,
) error {

	_, err := r.db.Exec(
		ctx,
		`UPDATE notifications
		 SET is_read = TRUE
		 WHERE user_id = $1 AND is_read = FALSE`,
		userID,
	)
	return err
}

// GetActiveStaffIDs mengambil ID user aktif dengan role kasir/admin
// untuk broadcast notifikasi.
func (r *NotificationRepository) GetActiveStaffIDs(
	ctx context.Context,
) ([]string, error) {

	rows, err := r.db.Query(
		ctx,
		`SELECT id::text
		 FROM users
		 WHERE role IN ('admin', 'kasir')
		   AND is_active = TRUE
		 ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]string, 0)

	for rows.Next() {
		var id string

		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}
