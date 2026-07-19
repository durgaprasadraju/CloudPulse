package phonepe_test

import (
	"context"
	"strings"
	"testing"

	"ride-sharing/services/payment-service/internal/infrastructure/phonepe"
	"ride-sharing/services/payment-service/pkg/types"
)

func TestLocalSessionIDPrefix(t *testing.T) {
	client := phonepe.NewLocalClient(&types.PhonePeConfig{})
	session, err := client.CreatePaymentSession(context.Background(), 15000, "INR", map[string]string{
		"trip_id": "t1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(session.SessionID, "pp_test_local_") {
		t.Fatalf("unexpected session id: %s", session.SessionID)
	}
	if session.Provider != types.ProviderPhonePe {
		t.Fatalf("provider want phonepe got %s", session.Provider)
	}
}

func TestIsConfiguredRequiresCredentials(t *testing.T) {
	if phonepe.IsConfigured(nil) {
		t.Fatal("nil should be unconfigured")
	}
	if phonepe.IsConfigured(&types.PhonePeConfig{}) {
		t.Fatal("empty should be unconfigured")
	}
	if phonepe.IsConfigured(&types.PhonePeConfig{MerchantID: "replace_me", SaltKey: "replace_me"}) {
		t.Fatal("placeholders should be unconfigured")
	}
	if !phonepe.IsConfigured(&types.PhonePeConfig{MerchantID: "MERCHANT123", SaltKey: "saltkeyvalue"}) {
		t.Fatal("real credentials should be configured")
	}
}
