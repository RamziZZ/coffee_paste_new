package fcm

import (
	"context"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// Client membungkus Firebase Cloud Messaging. Bisa dalam keadaan
// dinonaktifkan (tanpa credentials) supaya aplikasi tetap jalan
// dan push notification hanya dilewati dengan log.
type Client struct {
	enabled bool
	messaging *messaging.Client
}

// New membuat FCM client dari path file service account JSON.
// Jika credentialsPath kosong atau file tidak bisa dimuat, client
// dikembalikan dalam keadaan disabled (bukan error fatal).
func New(ctx context.Context, credentialsPath string) *Client {
	if credentialsPath == "" {
		log.Println("FCM dinonaktifkan: FIREBASE_CREDENTIALS_FILE belum diisi")
		return &Client{enabled: false}
	}

	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		log.Printf("FCM dinonaktifkan: gagal inisialisasi Firebase app: %v", err)
		return &Client{enabled: false}
	}

	messagingClient, err := app.Messaging(ctx)
	if err != nil {
		log.Printf("FCM dinonaktifkan: gagal inisialisasi Messaging client: %v", err)
		return &Client{enabled: false}
	}

	log.Println("FCM aktif: push notification siap dikirim")
	return &Client{enabled: true, messaging: messagingClient}
}

// Enabled melaporkan apakah FCM siap dipakai.
func (c *Client) Enabled() bool {
	return c != nil && c.enabled
}

// SendToTokens mengirim push notification ke banyak token sekaligus
// (multicast). Token yang sudah tidak terdaftar dihapus dari hasil
// dan dikembalikan supaya pemanggil bisa membersihkan database.
// Best-effort: kegagalan hanya dicatat di log.
func (c *Client) SendToTokens(
	ctx context.Context,
	tokens []string,
	title string,
	body string,
	data map[string]string,
) {
	if !c.Enabled() || len(tokens) == 0 {
		return
	}

	msg := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	resp, err := c.messaging.SendEachForMulticast(ctx, msg)
	if err != nil {
		log.Printf("FCM: gagal mengirim push ke %d token: %v", len(tokens), err)
		return
	}

	if resp.FailureCount > 0 {
		log.Printf(
			"FCM: %d/%d push gagal (failure_count=%d)",
			resp.FailureCount, len(tokens), resp.FailureCount,
		)
		for i, result := range resp.Responses {
			if !result.Success {
				log.Printf("FCM: token index %d gagal: %v", i, result.Error)
			}
		}
	}
}
