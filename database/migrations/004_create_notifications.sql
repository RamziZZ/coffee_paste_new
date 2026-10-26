-- Migration: Create notifications table
-- Jalankan: psql -U postgres -d coffee_paste_new -f database/migrations/004_create_notifications.sql

CREATE TABLE IF NOT EXISTS notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id),
    title      VARCHAR(150) NOT NULL,
    body       TEXT,
    type       VARCHAR(30) NOT NULL
               CHECK (type IN ('order_created', 'order_status', 'payment_paid', 'stock_low')),
    ref_id     TEXT,
    is_read    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index untuk pencarian berdasarkan user_id
CREATE INDEX IF NOT EXISTS idx_notifications_user_id
    ON notifications (user_id);

-- Index parsial untuk menghitung notifikasi unread dengan cepat
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread
    ON notifications (user_id)
    WHERE is_read = FALSE;

-- Index untuk laporan/riwayat berdasarkan waktu
CREATE INDEX IF NOT EXISTS idx_notifications_created_at
    ON notifications (created_at);

-- Comment
COMMENT ON TABLE notifications IS 'Notifikasi in-app untuk customer dan staff (kasir/admin)';
COMMENT ON COLUMN notifications.user_id IS 'Foreign key ke tabel users (penerima notifikasi)';
COMMENT ON COLUMN notifications.title IS 'Judul notifikasi';
COMMENT ON COLUMN notifications.body IS 'Isi/badan notifikasi (opsional)';
COMMENT ON COLUMN notifications.type IS 'Tipe notifikasi: order_created/order_status/payment_paid/stock_low';
COMMENT ON COLUMN notifications.ref_id IS 'ID terkait (misal order_id) untuk deep-link frontend';
COMMENT ON COLUMN notifications.is_read IS 'Status sudah dibaca';
