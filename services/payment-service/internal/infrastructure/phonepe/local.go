package phonepe

import (
	"context"
	"fmt"
	"log"

	"ride-sharing/services/payment-service/internal/domain"
	"ride-sharing/services/payment-service/pkg/types"

	"github.com/google/uuid"
)

type localClient struct {
	config *types.PhonePeConfig
}

func NewLocalClient(config *types.PhonePeConfig) domain.PaymentProcessor {
	log.Println("Using local PhonePe mock payment processor (no real charges)")
	return &localClient{config: config}
}

func (c *localClient) CreatePaymentSession(ctx context.Context, amount int64, currency string, metadata map[string]string) (*types.PaymentSession, error) {
	sessionID := fmt.Sprintf("pp_test_local_%s", uuid.New().String())
	log.Printf("Created local PhonePe session %s amount=%d %s metadata=%v", sessionID, amount, currency, metadata)
	return &types.PaymentSession{
		SessionID:   sessionID,
		Provider:    types.ProviderPhonePe,
		CheckoutURL: "", // rider uses in-app mock modal
	}, nil
}

// IsConfigured reports whether real PhonePe merchant credentials are present.
func IsConfigured(cfg *types.PhonePeConfig) bool {
	if cfg == nil {
		return false
	}
	if cfg.MerchantID == "" || cfg.SaltKey == "" {
		return false
	}
	if containsPlaceholder(cfg.MerchantID) || containsPlaceholder(cfg.SaltKey) {
		return false
	}
	return true
}

func containsPlaceholder(s string) bool {
	return s == "" || s == "replace_me" || len(s) < 4
}
