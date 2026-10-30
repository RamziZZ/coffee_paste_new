package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FCMTokenRepository mengelola token FCM per device untuk push notification.
type FCMTokenRepository struct {
	db *pgxpool.Pool
}

func NewFCMTokenRepository(db *pgxpool.Pool) *FCMTokenRepository {
	return &FCMTokenRepository{db: db}
}

// Upsert mendaftarkan atau memperbarui token FCM milik user.
// Jika token sudah ada (milik user mana pun), token dipindahkan ke user ini.
func (r *FCMTokenRepository) Upsert(
	ctx context.Context,
	userID string,
	fcmToken string,
	deviceType string,
) error {

	_, err := r.db.Exec(
		ctx,
		`INSERT INTO fcm_device_tokens (user_id, fcm_token, device_type)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (fcm_token)
		 DO UPDATE SET
			user_id = EXCLUDED.user_id,
			device_type = EXCLUDED.device_type,
			updated_at = NOW()`,
		userID,
		fcmToken,
		deviceType,
	)
	return err
}

// Delete menghapus token FCM milik user yang login.
func (r *FCMTokenRepository) Delete(
	ctx context.Context,
	userID string,
	fcmToken string,
) error {

	_, err := r.db.Exec(
		ctx,
		`DELETE FROM fcm_device_tokens
		 WHERE user_id = $1 AND fcm_token = $2`,
		userID,
		fcmToken,
	)
	return err
}

// GetTokensByUserIDs mengambil semua token aktif milik daftar user ID
// untuk keperluan push notification multicast.
func (r *FCMTokenRepository) GetTokensByUserIDs(
	ctx context.Context,
	userIDs []string,
) ([]string, error) {

	if len(userIDs) == 0 {
		return nil, nil
	}

	rows, err := r.db.Query(
		ctx,
		`SELECT fcm_token
		 FROM fcm_device_tokens
		 WHERE user_id = ANY($1)`,
		userIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := make([]string, 0)

	for rows.Next() {
		var token string

		if err := rows.Scan(&token); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}

// DeleteInvalidTokens menghapus token yang sudah tidak terdaftar di FCM.
func (r *FCMTokenRepository) DeleteInvalidTokens(
	ctx context.Context,
	tokens []string,
) error {

	if len(tokens) == 0 {
		return nil
	}

	_, err := r.db.Exec(
		ctx,
		`DELETE FROM fcm_device_tokens
		 WHERE fcm_token = ANY($1)`,
		tokens,
	)
	return err
}
