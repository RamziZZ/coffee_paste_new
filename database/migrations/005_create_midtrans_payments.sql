-- Migration: Create midtrans_payments table (skema coffee_paste_new, uuid)
-- Jalankan: psql -U postgres -d coffee_paste_new -f database/migrations/005_create_midtrans_payments.sql

-- Perluas enum pembayaran untuk status Midtrans (expired/refunded)
-- supaya gateway bisa mencatat status lengkap tanpa memaksa 'failed'.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e
        JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'payment_status' AND e.enumlabel = 'expired'
    ) THEN
        ALTER TYPE payment_status ADD VALUE 'expired';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e
        JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'payment_status' AND e.enumlabel = 'refunded'
    ) THEN
        ALTER TYPE payment_status ADD VALUE 'refunded';
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS midtrans_payments (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id          UUID NOT NULL REFERENCES orders(id),
    midtrans_trans_id VARCHAR(100),
    snap_token        TEXT,
    redirect_url      TEXT,
    amount            INTEGER NOT NULL,
    status            VARCHAR(20) NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'success', 'failed', 'expired', 'refunded')),
    payment_type      VARCHAR(50),
    bank              VARCHAR(30),
    masked_card       VARCHAR(30),
    fraud_status      VARCHAR(20),
    paid_at           TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
COMMENT ON COLUMN midtrans_payments.status IS 'Status pembayaran: pending/success/failed/expired/refunded';
