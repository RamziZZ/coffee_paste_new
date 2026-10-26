-- Migration: Create inventory_stocks table
-- Jalankan: psql -U postgres -d coffee_paste_new -f database/migrations/002_create_inventory_stocks.sql

CREATE TABLE IF NOT EXISTS inventory_stocks (
    id                  BIGSERIAL PRIMARY KEY,
    menu_id             UUID NOT NULL UNIQUE REFERENCES menus(id),
    qty                 INTEGER NOT NULL DEFAULT 0 CHECK (qty >= 0),
    low_stock_threshold INTEGER NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index untuk pencarian berdasarkan menu_id (unik sudah via UNIQUE constraint)
CREATE INDEX IF NOT EXISTS idx_inventory_stocks_menu_id
    ON inventory_stocks (menu_id);

-- Index untuk pencarian stok menipis
CREATE INDEX IF NOT EXISTS idx_inventory_stocks_low
    ON inventory_stocks (qty)
    WHERE qty <= low_stock_threshold;

-- Comment
COMMENT ON TABLE inventory_stocks IS 'Stok produk jadi (menus). Menu tanpa baris di tabel ini dianggap stok tidak dilacak (unlimited).';
COMMENT ON COLUMN inventory_stocks.menu_id IS 'Foreign key unik ke tabel menus';
COMMENT ON COLUMN inventory_stocks.qty IS 'Jumlah stok tersisa';
COMMENT ON COLUMN inventory_stocks.low_stock_threshold IS 'Batas minimum stok untuk memicu notifikasi stok menipis';
