package service

import (
	"context"
	"log"
	"time"

	"coffee_paste_backend/internal/notifications/dto"
	"coffee_paste_backend/internal/notifications/model"
	"coffee_paste_backend/internal/notifications/repository"
	"coffee_paste_backend/pkg/fcm"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

type NotificationService struct {
	repository      *repository.NotificationRepository
	fcmTokenRepo    *repository.FCMTokenRepository
	fcmClient       *fcm.Client
}

func NewNotificationService(
	repository *repository.NotificationRepository,
	fcmTokenRepo *repository.FCMTokenRepository,
	fcmClient *fcm.Client,
) *NotificationService {
	return &NotificationService{
		repository:   repository,
		fcmTokenRepo: fcmTokenRepo,
		fcmClient:    fcmClient,
	}
}

func toNotificationResponse(n model.Notification) dto.NotificationResponse {
	return dto.NotificationResponse{
		ID:        n.ID,
		Title:     n.Title,
		Body:      n.Body,
		Type:      n.Type,
		RefID:     n.RefID,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt,
	}
}

// Notify mengirim notifikasi in-app ke satu user. Best-effort:
// kegagalan hanya dicatat di log supaya operasi utama pemanggil
// tidak ikut gagal.
func (s *NotificationService) Notify(
	ctx context.Context,
	userID string,
	notifType string,
	refID string,
	title string,
	body string,
) {
	if userID == "" {
		return
	}

	var bodyPtr *string
	if body != "" {
		bodyPtr = &body
	}

	var refPtr *string
	if refID != "" {
		refPtr = &refID
	}

	err := s.repository.Create(ctx, model.NewNotification{
		UserID: userID,
		Title:  title,
		Body:   bodyPtr,
		Type:   notifType,
		RefID:  refPtr,
	})
	if err != nil {
		log.Printf(
			"Gagal membuat notifikasi user %s (tipe %s): %v",
			userID, notifType, err,
		)
	}

	// Push notification via FCM (best-effort, async)
	s.pushToUsers([]string{userID}, title, body, notifType, refID)
}

// NotifyStaff mengirim notifikasi ke semua user aktif role kasir/admin.
// Best-effort, sama seperti Notify.
func (s *NotificationService) NotifyStaff(
	ctx context.Context,
	notifType string,
	refID string,
	title string,
	body string,
) {
	staffIDs, err := s.repository.GetActiveStaffIDs(ctx)
	if err != nil {
		log.Printf("Gagal mengambil daftar staff untuk notifikasi: %v", err)
		return
	}

	if len(staffIDs) == 0 {
		return
	}

	var bodyPtr *string
	if body != "" {
		bodyPtr = &body
	}

	var refPtr *string
	if refID != "" {
		refPtr = &refID
	}

	notifications := make([]model.NewNotification, 0, len(staffIDs))
	for _, id := range staffIDs {
		notifications = append(notifications, model.NewNotification{
			UserID: id,
			Title:  title,
			Body:   bodyPtr,
			Type:   notifType,
			RefID:  refPtr,
		})
	}

	if err := s.repository.CreateBatch(ctx, notifications); err != nil {
		log.Printf("Gagal broadcast notifikasi ke staff: %v", err)
	}

	// Push notification via FCM (best-effort, async)
	s.pushToUsers(staffIDs, title, body, notifType, refID)
}

// pushToUsers mengirim push notification FCM ke daftar user.
// Best-effort: kegagalan hanya dicatat di log.
func (s *NotificationService) pushToUsers(
	userIDs []string,
	title string,
	body string,
	notifType string,
	refID string,
) {
	if s.fcmClient == nil || !s.fcmClient.Enabled() {
		return
	}

	if len(userIDs) == 0 {
		return
	}

	go func() {
		// Konteks timeout terpisah karena goroutine tidak
		// boleh memakai konteks request yang sudah selesai.
		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		tokens, err := s.fcmTokenRepo.GetTokensByUserIDs(ctx, userIDs)
		if err != nil {
			log.Printf("FCM: gagal mengambil device tokens: %v", err)
			return
		}

		if len(tokens) == 0 {
			return
		}

		data := map[string]string{
			"type": notifType,
		}
		if refID != "" {
			data["ref_id"] = refID
		}

		s.fcmClient.SendToTokens(ctx, tokens, title, body, data)
	}()
}

// List mengambil daftar notifikasi milik user yang login.
func (s *NotificationService) List(
	ctx context.Context,
	userID string,
	q dto.ListNotificationsQuery,
) ([]dto.NotificationResponse, dto.Meta, error) {

	page := q.Page
	if page < 1 {
		page = 1
	}

	limit := q.Limit
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	notifications, total, err := s.repository.List(ctx, model.ListFilter{
		UserID: userID,
		Limit:  limit,
		Offset: (page - 1) * limit,
	})
	if err != nil {
		return nil, dto.Meta{}, err
	}

	unread, err := s.repository.CountUnread(ctx, userID)
	if err != nil {
		return nil, dto.Meta{}, err
	}

	result := make([]dto.NotificationResponse, 0, len(notifications))
	for _, n := range notifications {
		result = append(result, toNotificationResponse(n))
	}

	return result, dto.Meta{
		Page:        page,
		Limit:       limit,
		Total:       total,
		UnreadCount: unread,
	}, nil
}

// UnreadCount menghitung notifikasi yang belum dibaca.
func (s *NotificationService) UnreadCount(
	ctx context.Context,
	userID string,
) (int, error) {
	return s.repository.CountUnread(ctx, userID)
}

// MarkRead menandai satu notifikasi sebagai sudah dibaca.
func (s *NotificationService) MarkRead(
	ctx context.Context,
	userID string,
	notificationID string,
) error {
	return s.repository.MarkRead(ctx, userID, notificationID)
}

// MarkAllRead menandai semua notifikasi sebagai sudah dibaca.
func (s *NotificationService) MarkAllRead(
	ctx context.Context,
	userID string,
) error {
	return s.repository.MarkAllRead(ctx, userID)
}

// RegisterDeviceToken mendaftarkan token FCM milik user yang login.
func (s *NotificationService) RegisterDeviceToken(
	ctx context.Context,
	userID string,
	req dto.RegisterDeviceTokenRequest,
) error {
	return s.fcmTokenRepo.Upsert(ctx, userID, req.FCMToken, req.DeviceType)
}

// DeleteDeviceToken menghapus token FCM milik user yang login
// (dipakai saat logout / unregister notifikasi di client).
func (s *NotificationService) DeleteDeviceToken(
	ctx context.Context,
	userID string,
	fcmToken string,
) error {
	return s.fcmTokenRepo.Delete(ctx, userID, fcmToken)
}
