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

// localClient creates fake checkout session IDs for local prototypes
// when real Stripe keys are not configured.
type localClient struct {
	config *types.PaymentConfig
}

func NewLocalClient(config *types.PaymentConfig) domain.PaymentProcessor {
	log.Println("Using local mock payment processor (no real Stripe charges)")
	return &localClient{config: config}
}

func (c *localClient) CreatePaymentSession(ctx context.Context, amount int64, currency string, metadata map[string]string) (string, error) {
	sessionID := fmt.Sprintf("cs_test_local_%s", uuid.New().String())
	log.Printf("Created local payment session %s amount=%d %s metadata=%v", sessionID, amount, currency, metadata)
	return sessionID, nil
}

// IsLocalStripeKey reports whether the configured secret is a placeholder / local-only key.
func IsLocalStripeKey(secret string) bool {
	if secret == "" || strings.Contains(secret, "replace_me") {
		return true
	}
	return strings.HasPrefix(secret, "sk_test_local")
}
