package midtrans

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	sandboxBaseURL    = "https://app.sandbox.midtrans.com"
	productionBaseURL = "https://app.midtrans.com"

	snapPath = "/snap/v1/transactions"
)

type Client struct {
	serverKey    string
	merchantName string
	baseURL      string
	httpClient   *http.Client
}

func NewClient(serverKey, merchantName, environment string) *Client {
	baseURL := sandboxBaseURL
	if environment == "production" {
		baseURL = productionBaseURL
	}

	return &Client{
		serverKey:    serverKey,
		merchantName: merchantName,
		baseURL:      baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ItemDetail adalah satu baris item dalam transaksi Midtrans.
type ItemDetail struct {
	ID       string `json:"id"`
	Price    int    `json:"price"`
	Qty      int    `json:"qty"`
	Name     string `json:"name"`
}

// CustomerDetail berisi data pelanggan untuk Midtrans.
type CustomerDetail struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
}

// TransactionDetails adalah detail transaksi untuk Midtrans.
type TransactionDetails struct {
	OrderID     string `json:"order_id"`
	GrossAmount int    `json:"gross_amount"`
}

// SnapRequest adalah body request untuk membuat Snap transaction.
type SnapRequest struct {
	TransactionDetails TransactionDetails `json:"transaction_details"`
	ItemDetails        []ItemDetail       `json:"item_details,omitempty"`
	CustomerDetails    *CustomerDetail    `json:"customer_details,omitempty"`
	Expiry             *Expiry            `json:"expiry,omitempty"`
	Callbacks          *Callbacks         `json:"callbacks,omitempty"`
}

// Expiry mengatur waktu kedaluwarsa transaksi.
type Expiry struct {
	Unit     string `json:"unit"`
	Duration int    `json:"duration"`
}

// Callbacks berisi URL callback Midtrans.
type Callbacks struct {
	Finish string `json:"finish,omitempty"`
}

// SnapResponse adalah response dari Midtrans Snap API.
type SnapResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

// Notification adalah payload notifikasi (webhook) dari Midtrans.
type Notification struct {
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"`
	TransactionID     string `json:"transaction_id"`
	StatusMessage     string `json:"status_message"`
	SignatureKey      string `json:"signature_key"`
	PaymentType       string `json:"payment_type"`
	OrderID           string `json:"order_id"`
	MerchantID        string `json:"merchant_id"`
	GrossAmount       string `json:"gross_amount"`
	FraudStatus       string `json:"fraud_status"`
	StatusCode        string `json:"status_code"`
	Currency          string `json:"currency"`
	Bank              string `json:"bank,omitempty"`
	CardType          string `json:"card_type,omitempty"`
	MaskedCard        string `json:"masked_card,omitempty"`
}

// CreateSnapTransaction membuat transaksi Snap baru di Midtrans.
func (c *Client) CreateSnapTransaction(req SnapRequest) (*SnapResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("gagal encode request: %w", err)
	}

	url := c.baseURL + snapPath
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gagal membuat request: %w", err)
	}

	httpReq.SetBasicAuth(c.serverKey, "")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gagal memanggil Midtrans: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Midtrans mengembalikan status %d: %s", resp.StatusCode, string(respBody))
	}

	var snapResp SnapResponse
	if err := json.Unmarshal(respBody, &snapResp); err != nil {
		return nil, fmt.Errorf("gagal decode response: %w", err)
	}

	return &snapResp, nil
}

// VerifyNotificationSignature memverifikasi signature key dari notifikasi Midtrans.
// signature = SHA512(order_id + status_code + gross_amount + server_key)
func (c *Client) VerifyNotificationSignature(n Notification) bool {
	data := n.OrderID + n.StatusCode + n.GrossAmount + c.serverKey
	hash := sha512.Sum512([]byte(data))
	expected := hex.EncodeToString(hash[:])

	return expected == n.SignatureKey
}

// StatusMapping memetakan transaction_status Midtrans ke status pembayaran lokal.
//   - settlement, capture → paid
//   - pending             → pending
//   - deny, cancel        → failed
//   - expire              → expired
//   - refund              → refunded
func StatusMapping(transactionStatus, fraudStatus string) string {
	switch transactionStatus {
	case "settlement":
		return "paid"
	case "capture":
		if fraudStatus == "accept" {
			return "paid"
		}
		return "pending"
	case "pending":
		return "pending"
	case "deny", "cancel":
		return "failed"
	case "expire":
		return "expired"
	case "refund", "partial_refund":
		return "refunded"
	default:
		return "pending"
	}
}
