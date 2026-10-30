-- Migration: Tambah kolom google_id & tabel fcm_device_tokens
-- Jalankan: psql -U postgres -d coffee_paste_new -f database/migrations/006_google_fcm.sql

-- 1. Kolom google_id pada users (untuk login via Google, subject Google = sub)
ALTER TABLE users ADD COLUMN IF NOT EXISTS google_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_google_id
    ON users (google_id)
    WHERE google_id IS NOT NULL;

-- password_hash dibuat nullable agar user Google (tanpa password) bisa dibuat
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

-- 2. Tabel device token FCM
CREATE TABLE IF NOT EXISTS fcm_device_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fcm_token   TEXT NOT NULL,
    device_type VARCHAR(20) NOT NULL DEFAULT 'android',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_fcm_device_tokens_token
    ON fcm_device_tokens (fcm_token);

CREATE INDEX IF NOT EXISTS idx_fcm_device_tokens_user_id
    ON fcm_device_tokens (user_id);

-- Comment
COMMENT ON COLUMN users.google_id IS 'Google OAuth subject (sub) untuk login dengan Google';
COMMENT ON TABLE fcm_device_tokens IS 'Token FCM per device untuk push notification';
COMMENT ON COLUMN fcm_device_tokens.fcm_token IS 'Registration token dari Firebase Messaging';
COMMENT ON COLUMN fcm_device_tokens.device_type IS 'android / ios / web';
