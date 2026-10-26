package service

import (
	"context"
	"log"

	"coffee_paste_backend/internal/notifications/dto"
	"coffee_paste_backend/internal/notifications/model"
	"coffee_paste_backend/internal/notifications/repository"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

type NotificationService struct {
	repository *repository.NotificationRepository
}

func NewNotificationService(
	repository *repository.NotificationRepository,
) *NotificationService {
	return &NotificationService{
		repository: repository,
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
