package phonepe

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"ride-sharing/services/payment-service/internal/domain"
	"ride-sharing/services/payment-service/pkg/types"

	"github.com/google/uuid"
)

const (
	uatHost  = "https://api-preprod.phonepe.com/apis/pg-sandbox"
	prodHost = "https://api.phonepe.com/apis/hermes"
)

type client struct {
	config *types.PhonePeConfig
	http   *http.Client
}

func NewClient(config *types.PhonePeConfig) domain.PaymentProcessor {
	log.Println("Using PhonePe payment processor")
	return &client{
		config: config,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *client) baseURL() string {
	if strings.EqualFold(c.config.Env, "PROD") {
		return prodHost
	}
	return uatHost
}

func (c *client) CreatePaymentSession(ctx context.Context, amount int64, currency string, metadata map[string]string) (*types.PaymentSession, error) {
	merchantTxnID := "TXN" + strings.ReplaceAll(uuid.New().String(), "-", "")[:20]
	payload := map[string]any{
		"merchantId":            c.config.MerchantID,
		"merchantTransactionId": merchantTxnID,
		"merchantUserId":        metadata["user_id"],
		"amount":                amount, // paise
		"redirectUrl":           c.config.SuccessURL,
		"redirectMode":          "REDIRECT",
		"callbackUrl":           c.config.CallbackURL,
		"paymentInstrument": map[string]string{
			"type": "PAY_PAGE",
		},
		"metaInfo": map[string]string{
			"udF1": metadata["trip_id"],
			"udF2": metadata["user_id"],
			"udF3": metadata["driver_id"],
		},
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	b64 := base64.StdEncoding.EncodeToString(raw)
	path := "/pg/v1/pay"
	checksum := sha256Hex(b64 + path + c.config.SaltKey) + "###" + c.config.SaltIndex

	body, _ := json.Marshal(map[string]string{"request": b64})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL()+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-VERIFY", checksum)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("phonepe request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var parsed struct {
		Success bool `json:"success"`
		Code    string `json:"code"`
		Data    struct {
			InstrumentResponse struct {
				RedirectInfo struct {
					URL string `json:"url"`
				} `json:"redirectInfo"`
			} `json:"instrumentResponse"`
			MerchantTransactionID string `json:"merchantTransactionId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("phonepe decode failed: %w body=%s", err, string(respBody))
	}
	if !parsed.Success || parsed.Data.InstrumentResponse.RedirectInfo.URL == "" {
		return nil, fmt.Errorf("phonepe pay failed: %s", string(respBody))
	}

	sessionID := parsed.Data.MerchantTransactionID
	if sessionID == "" {
		sessionID = merchantTxnID
	}
	return &types.PaymentSession{
		SessionID:   sessionID,
		Provider:    types.ProviderPhonePe,
		CheckoutURL: parsed.Data.InstrumentResponse.RedirectInfo.URL,
	}, nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// VerifyCallbackChecksum validates X-VERIFY for PhonePe server callbacks.
func VerifyCallbackChecksum(b64Body, saltKey, saltIndex, xVerify string) bool {
	expected := sha256Hex(b64Body+saltKey) + "###" + saltIndex
	return strings.EqualFold(expected, xVerify)
}
