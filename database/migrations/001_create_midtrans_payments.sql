-- Migration: Create midtrans_payments table
-- Jalankan: psql -U postgres -d coffee_paste_pos -f database/migrations/001_create_midtrans_payments.sql

CREATE TABLE IF NOT EXISTS midtrans_payments (
    id              BIGSERIAL PRIMARY KEY,
    order_id        UUID NOT NULL REFERENCES orders(id),
    midtrans_trans_id VARCHAR(100),
    snap_token      TEXT,
    redirect_url    TEXT,
    amount          INTEGER NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'paid', 'failed', 'expired', 'refunded')),
    payment_type    VARCHAR(50),
    bank            VARCHAR(30),
    masked_card     VARCHAR(30),
    fraud_status    VARCHAR(20),
    paid_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index untuk pencarian berdasarkan order_id
CREATE INDEX IF NOT EXISTS idx_midtrans_payments_order_id
    ON midtrans_payments (order_id);

-- Index untuk pencarian berdasarkan midtrans transaction ID
CREATE INDEX IF NOT EXISTS idx_midtrans_payments_midtrans_trans_id
    ON midtrans_payments (midtrans_trans_id)
    WHERE midtrans_trans_id IS NOT NULL;

-- Index untuk pencarian berdasarkan status
CREATE INDEX IF NOT EXISTS idx_midtrans_payments_status
    ON midtrans_payments (status);

-- Comment
COMMENT ON TABLE midtrans_payments IS 'Pembayaran melalui Midtrans payment gateway';
COMMENT ON COLUMN midtrans_payments.order_id IS 'Foreign key ke tabel orders';
COMMENT ON COLUMN midtrans_payments.midtrans_trans_id IS 'Transaction ID dari Midtrans';
COMMENT ON COLUMN midtrans_payments.snap_token IS 'Snap token untuk frontend Midtrans Snap.js';
COMMENT ON COLUMN midtrans_payments.redirect_url IS 'URL redirect ke halaman pembayaran Midtrans';
COMMENT ON COLUMN midtrans_payments.amount IS 'Jumlah pembayaran dalam Rupiah';
COMMENT ON COLUMN midtrans_payments.status IS 'Status pembayaran: pending/paid/failed/expired/refunded';
