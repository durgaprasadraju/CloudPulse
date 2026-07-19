package otp

import (
	"testing"
	"time"
)

func TestGenerateLength(t *testing.T) {
	code, err := Generate(6)
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6 digits, got %q", code)
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Fatalf("non-digit in OTP: %q", code)
		}
	}
}

func TestHashAndVerify(t *testing.T) {
	code := "482913"
	hash := Hash(code)
	if hash == code {
		t.Fatal("hash must not equal plaintext")
	}
	if !Verify(code, hash) {
		t.Fatal("expected verify to succeed")
	}
	if Verify("000000", hash) {
		t.Fatal("wrong OTP must fail")
	}
	if Verify("", hash) {
		t.Fatal("empty OTP must fail")
	}
}

func TestDefaultTTL(t *testing.T) {
	if DefaultTTL < time.Minute {
		t.Fatal("TTL too short")
	}
	if DefaultMaxAttempts < 3 {
		t.Fatal("max attempts too low")
	}
}
