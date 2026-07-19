package stripe

import (
	"context"
	"fmt"
	"log"
	"strings"

	"ride-sharing/services/payment-service/internal/domain"
	"ride-sharing/services/payment-service/pkg/types"

	"github.com/google/uuid"
)

type localClient struct {
	config *types.PaymentConfig
}

func NewLocalClient(config *types.PaymentConfig) domain.PaymentProcessor {
	log.Println("Using local mock Stripe payment processor (no real charges)")
	return &localClient{config: config}
}

func (c *localClient) CreatePaymentSession(ctx context.Context, amount int64, currency string, metadata map[string]string) (*types.PaymentSession, error) {
	sessionID := fmt.Sprintf("cs_test_local_%s", uuid.New().String())
	log.Printf("Created local Stripe session %s amount=%d %s metadata=%v", sessionID, amount, currency, metadata)
	return &types.PaymentSession{
		SessionID: sessionID,
		Provider:  types.ProviderStripe,
	}, nil
}

func IsLocalStripeKey(secret string) bool {
	if secret == "" || strings.Contains(secret, "replace_me") {
		return true
	}
	return strings.HasPrefix(secret, "sk_test_local")
}
