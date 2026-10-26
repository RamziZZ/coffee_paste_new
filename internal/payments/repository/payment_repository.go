package repository

import (
	"context"
	"errors"

	"coffee_paste_backend/internal/payments/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrOrderNotFound      = errors.New("pesanan tidak ditemukan")
	ErrPaymentNotFound    = errors.New("pembayaran tidak ditemukan")
	ErrOrderAlreadyPaid   = errors.New("pesanan sudah dibayar")
	ErrAmountNotMatch     = errors.New("jumlah pembayaran harus sama dengan total pesanan")
	ErrCashBelowAmount    = errors.New("uang tunai yang diterima kurang dari jumlah pembayaran")
	ErrInvalidTransition  = errors.New("peralihan status pembayaran tidak diizinkan")
	ErrCashNotApplicable  = errors.New("cash_received hanya untuk pembayaran tunai")
	ErrNoChangeForNonCash = errors.New("change_amount hanya dihitung untuk pembayaran tunai")
)

type PaymentRepository struct {
	DB *pgxpool.Pool
}

func NewPaymentRepository(db *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{DB: db}
}

const paymentSelect = `
	SELECT
		id::text,
		order_id::text,
		received_by::text,
		method::text,
		amount,
		cash_received,
		change_amount,
		status::text,
		reference_no,
		paid_at
	FROM payments
`

func scanPayment(row pgx.Row) (*model.Payment, error) {
	var payment model.Payment

	err := row.Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.ReceivedBy,
		&payment.Method,
		&payment.Amount,
		&payment.CashReceived,
		&payment.ChangeAmount,
		&payment.Status,
		&payment.ReferenceNo,
		&payment.PaidAt,
	)
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

// orderTotal menghitung total pesanan dari order_items dikurangi diskon,
// sambil mengunci baris orders (FOR UPDATE) agar tidak double-bayar.
func orderTotal(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (int, error) {
	var total int

	err := tx.QueryRow(ctx, `
		SELECT COALESCE(
			(
				SELECT SUM(oi.price * oi.qty)
				FROM order_items oi
				WHERE oi.order_id = o.id
			),
			0
		) - o.discount
		FROM orders o
		WHERE o.id = $1
		FOR UPDATE
	`, orderID).Scan(&total)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrOrderNotFound
		}
		return 0, err
	}

	return total, nil
}

func (r *PaymentRepository) Create(
	ctx context.Context,
	payment *model.Payment,
) (*model.Payment, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	orderTotal, err := orderTotal(ctx, tx, payment.OrderID)
	if err != nil {
		return nil, err
	}

	var paid bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM payments
			WHERE order_id = $1 AND status = 'success'
		)
	`, payment.OrderID).Scan(&paid)
	if err != nil {
		return nil, err
	}
	if paid {
		return nil, ErrOrderAlreadyPaid
	}

	if payment.Amount != orderTotal {
		return nil, ErrAmountNotMatch
	}

	if payment.CashReceived != nil {
		if payment.Method != "tunai" {
			return nil, ErrCashNotApplicable
		}
		if *payment.CashReceived < payment.Amount {
			return nil, ErrCashBelowAmount
		}
		change := *payment.CashReceived - payment.Amount
		payment.ChangeAmount = &change
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO payments (
			id,
			order_id,
			received_by,
			method,
			amount,
			cash_received,
			change_amount,
			status,
			reference_no
		)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, 'pending', $7)
		RETURNING id::text
	`,
		payment.OrderID,
		payment.ReceivedBy,
		payment.Method,
		payment.Amount,
		payment.CashReceived,
		payment.ChangeAmount,
		payment.ReferenceNo,
	).Scan(&payment.ID)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return payment, nil
}

func (r *PaymentRepository) GetByID(
	ctx context.Context,
	id string,
) (*model.Payment, error) {
	payment, err := scanPayment(r.DB.QueryRow(
		ctx,
		paymentSelect+`WHERE id = $1`,
		id,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPaymentNotFound
	}
	if err != nil {
		return nil, err
	}

	return payment, nil
}

func (r *PaymentRepository) GetByOrderID(
	ctx context.Context,
	orderID string,
) ([]model.Payment, error) {
	rows, err := r.DB.Query(ctx, paymentSelect+`
		WHERE order_id = $1
		ORDER BY paid_at DESC NULLS LAST, id DESC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	payments := make([]model.Payment, 0)

	for rows.Next() {
		payment, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		payments = append(payments, *payment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return payments, nil
}

// allowedTransition menentukan apakah peralihan status diizinkan.
func allowedTransition(from, to string) bool {
	switch from {
	case "pending":
		return to == "success" || to == "failed" || to == "expired"
	case "success":
		return to == "refunded"
	default:
		// failed, expired, dan refunded adalah status terminal.
		return false
	}
}

func (r *PaymentRepository) UpdateStatus(
	ctx context.Context,
	id string,
	status string,
	cashReceived *int,
) (*model.Payment, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var payment model.Payment

	err = tx.QueryRow(ctx, paymentSelect+`
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.ReceivedBy,
		&payment.Method,
		&payment.Amount,
		&payment.CashReceived,
		&payment.ChangeAmount,
		&payment.Status,
		&payment.ReferenceNo,
		&payment.PaidAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPaymentNotFound
	}
	if err != nil {
		return nil, err
	}

	if status != payment.Status && !allowedTransition(payment.Status, status) {
		return nil, ErrInvalidTransition
	}

	if cashReceived != nil {
		if payment.Method != "tunai" {
			return nil, ErrCashNotApplicable
		}
		if *cashReceived < payment.Amount {
			return nil, ErrCashBelowAmount
		}
		change := *cashReceived - payment.Amount
		payment.ChangeAmount = &change
		payment.CashReceived = cashReceived
	}

	payment.Status = status

	if status == "success" && payment.PaidAt == nil {
		err = tx.QueryRow(ctx, `
			UPDATE payments
			SET
				status = 'success',
				cash_received = $2,
				change_amount = $3,
				paid_at = NOW()
			WHERE id = $1
			RETURNING paid_at
		`, id, payment.CashReceived, payment.ChangeAmount).Scan(&payment.PaidAt)
		if err != nil {
			return nil, err
		}
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE payments
			SET
				status = $2,
				cash_received = $3,
				change_amount = $4
			WHERE id = $1
		`, id, status, payment.CashReceived, payment.ChangeAmount)
		if err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &payment, nil
}
