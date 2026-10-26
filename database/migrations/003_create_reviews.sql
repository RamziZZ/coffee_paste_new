-- Migration: Create reviews table
-- Jalankan: psql -U postgres -d coffee_paste_new -f database/migrations/003_create_reviews.sql

CREATE TABLE IF NOT EXISTS reviews (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id   UUID NOT NULL UNIQUE REFERENCES orders(id),
    user_id    UUID NOT NULL REFERENCES users(id),
    rating     SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index untuk pencarian berdasarkan user_id
CREATE INDEX IF NOT EXISTS idx_reviews_user_id
    ON reviews (user_id);

-- Index untuk filter/rata-rata berdasarkan rating
CREATE INDEX IF NOT EXISTS idx_reviews_rating
    ON reviews (rating);

-- Index untuk laporan berdasarkan periode pembuatan
CREATE INDEX IF NOT EXISTS idx_reviews_created_at
    ON reviews (created_at);

-- Comment
COMMENT ON TABLE reviews IS 'Ulasan dan rating pelanggan. Satu review per pesanan yang berstatus selesai.';
COMMENT ON COLUMN reviews.order_id IS 'Foreign key unik ke tabel orders (satu review per pesanan)';
COMMENT ON COLUMN reviews.user_id IS 'Foreign key ke tabel users (pelanggan pemilik pesanan)';COMMENT ON COLUMN reviews.rating IS 'Rating 1 sampai 5';
COMMENT ON COLUMN reviews.comment IS 'Komentar ulasan (opsional)';
