package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"coffee_paste_backend/internal/payments/midtrans/model"
)

type MidtransRepository struct {
	DB *pgxpool.Pool
}

func NewMidtransRepository(db *pgxpool.Pool) *MidtransRepository {
	return &MidtransRepository{DB: db}
}

const midtransSelect = `
	SELECT
		id::text,
		order_id::text,
		midtrans_trans_id,
		snap_token,
		redirect_url,
		amount,
		status,
		payment_type,
		bank,
		masked_card,
		fraud_status,
		paid_at,
		created_at,
		updated_at
	FROM midtrans_payments
`

func scanMidtransPayment(row pgx.Row) (*model.MidtransPayment, error) {
	var payment model.MidtransPayment

	err := row.Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.MidtransTransID,
		&payment.SnapToken,
		&payment.RedirectURL,
		&payment.Amount,
		&payment.Status,
		&payment.PaymentType,
		&payment.Bank,
		&payment.MaskedCard,
		&payment.FraudStatus,
		&payment.PaidAt,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

// GetOrderInfo mengambil total (setelah diskon) dan status bayar sebuah pesanan.
// Status bayar ditentukan dari keberadaan pembayaran success di tabel payments.
func (r *MidtransRepository) GetOrderInfo(
	ctx context.Context,
	orderID string,
) (total int, paid bool, orderCode string, err error) {

	err = r.DB.QueryRow(ctx, `
		SELECT
			COALESCE(
				(
					SELECT SUM(oi.price * oi.qty)
					FROM order_items oi
					WHERE oi.order_id = o.id
				),
				0
			)::int - o.discount,
			EXISTS(
				SELECT 1 FROM payments p
				WHERE p.order_id = o.id AND p.status = 'success'
			),
			o.order_code
		FROM orders o
		WHERE o.id = $1
		FOR UPDATE
	`, orderID).Scan(&total, &paid, &orderCode)

	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, "", errors.New("pesanan tidak ditemukan")
	}
	if err != nil {
		return 0, false, "", err
	}

	return total, paid, orderCode, nil
}

// GetOrderItems mengambil item pesanan untuk dikirim ke Midtrans.
func (r *MidtransRepository) GetOrderItems(
	ctx context.Context,
	orderID string,
) ([]struct {
	MenuID   string
	MenuName string
	Price    int
	Qty      int
}, error) {

	rows, err := r.DB.Query(ctx, `
		SELECT menu_id::text, menu_name, price, qty
		FROM order_items
		WHERE order_id = $1
		ORDER BY menu_name
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]struct {
		MenuID   string
		MenuName string
		Price    int
		Qty      int
	}, 0)

	for rows.Next() {
		var it struct {
			MenuID   string
			MenuName string
			Price    int
			Qty      int
		}
		if err := rows.Scan(&it.MenuID, &it.MenuName, &it.Price, &it.Qty); err != nil {
			return nil, err
		}
		items = append(items, it)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

// GetCustomerInfo mengambil data customer dari pesanan.
func (r *MidtransRepository) GetCustomerInfo(
	ctx context.Context,
	orderID string,
) (name string, email *string, phone *string, err error) {

	err = r.DB.QueryRow(ctx, `
		SELECT COALESCE(c.name, ''), c.email, c.phone
		FROM orders o
		LEFT JOIN customers c ON c.id = o.customer_id
		WHERE o.id = $1
	`, orderID).Scan(&name, &email, &phone)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, nil, errors.New("pesanan tidak ditemukan")
	}
	if err != nil {
		return "", nil, nil, err
	}

	return name, email, phone, nil
}

// GetOrderNotifyInfo mengambil user_id customer pemilik pesanan dan
// kode pesanan untuk keperluan notifikasi in-app.
func (r *MidtransRepository) GetOrderNotifyInfo(
	ctx context.Context,
	orderID string,
) (customerUserID *string, orderCode string, err error) {

	err = r.DB.QueryRow(ctx, `
		SELECT u.id::text, o.order_code
		FROM orders o
		LEFT JOIN customers c ON c.id = o.customer_id
		LEFT JOIN users u ON u.id = c.user_id
		WHERE o.id = $1
	`, orderID).Scan(&customerUserID, &orderCode)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", errors.New("pesanan tidak ditemukan")
	}
	if err != nil {
		return nil, "", err
	}

	return customerUserID, orderCode, nil
}

// Create membuat record pembayaran Midtrans baru.
func (r *MidtransRepository) Create(
	ctx context.Context,
	payment *model.MidtransPayment,
) (*model.MidtransPayment, error) {

	err := r.DB.QueryRow(ctx, `
		INSERT INTO midtrans_payments (
			id, order_id, snap_token, redirect_url, amount, status
		)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)
		RETURNING
			id::text, order_id::text, midtrans_trans_id, snap_token,
			redirect_url, amount, status, payment_type, bank,
			masked_card, fraud_status, paid_at, created_at, updated_at
	`,
		payment.OrderID,
		payment.SnapToken,
		payment.RedirectURL,
		payment.Amount,
		payment.Status,
	).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.MidtransTransID,
		&payment.SnapToken,
		&payment.RedirectURL,
		&payment.Amount,
		&payment.Status,
		&payment.PaymentType,
		&payment.Bank,
		&payment.MaskedCard,
		&payment.FraudStatus,
		&payment.PaidAt,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return payment, nil
}

// GetByOrderID mengambil pembayaran Midtrans terbaru untuk pesanan tertentu.
func (r *MidtransRepository) GetByOrderID(
	ctx context.Context,
	orderID string,
) (*model.MidtransPayment, error) {

	payment, err := scanMidtransPayment(r.DB.QueryRow(ctx, midtransSelect+`
		WHERE order_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, orderID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return payment, nil
}

// GetByID mengambil pembayaran Midtrans berdasarkan ID.
func (r *MidtransRepository) GetByID(
	ctx context.Context,
	id string,
) (*model.MidtransPayment, error) {

	payment, err := scanMidtransPayment(r.DB.QueryRow(ctx, midtransSelect+`
		WHERE id = $1
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("pembayaran tidak ditemukan")
	}
	if err != nil {
		return nil, err
	}

	return payment, nil
}

// GetByMidtransTransID mengambil pembayaran berdasarkan Midtrans transaction ID.
func (r *MidtransRepository) GetByMidtransTransID(
	ctx context.Context,
	midtransTransID string,
) (*model.MidtransPayment, error) {

	payment, err := scanMidtransPayment(r.DB.QueryRow(ctx, midtransSelect+`
		WHERE midtrans_trans_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, midtransTransID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return payment, nil
}

// UpdateFromNotification memperbarui status pembayaran dari notifikasi Midtrans.
// status harus salah satu dari enum payment_status ('success', 'failed',
// 'expired', 'refunded'); status sudah dipetakan oleh service.
// Skema orders tidak punya kolom paid, jadi tidak ada update orders.
func (r *MidtransRepository) UpdateFromNotification(
	ctx context.Context,
	midtransTransID string,
	orderID string,
	status string,
	paymentType string,
	bank *string,
	maskedCard *string,
	fraudStatus string,
) (*model.MidtransPayment, error) {

	payment, err := scanMidtransPayment(r.DB.QueryRow(ctx, `
		UPDATE midtrans_payments
		SET
			status = $3,
			payment_type = COALESCE($4, payment_type),
			bank = COALESCE($5, bank),
			masked_card = COALESCE($6, masked_card),
			fraud_status = $7,
			midtrans_trans_id = COALESCE($2, midtrans_trans_id),
			paid_at = CASE
				WHEN $3 = 'success' THEN COALESCE(paid_at, NOW())
				ELSE paid_at
			END,
			updated_at = NOW()
		WHERE order_id = $1
		RETURNING
			id::text, order_id::text, midtrans_trans_id, snap_token,
			redirect_url, amount, status, payment_type, bank,
			masked_card, fraud_status, paid_at, created_at, updated_at
	`,
		orderID,
		midtransTransID,
		status,
		paymentType,
		bank,
		maskedCard,
		fraudStatus,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("pembayaran tidak ditemukan untuk pesanan ini")
	}
	if err != nil {
		return nil, err
	}

	return payment, nil
}
