package auth_test

import (
	"os"
	"testing"
	"time"

	"ride-sharing/shared/auth"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := auth.HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if !auth.CheckPassword(hash, "secret123") {
		t.Fatal("expected password to match")
	}
	if auth.CheckPassword(hash, "wrong") {
		t.Fatal("expected password mismatch")
	}
}

func TestIssueAndParseToken(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	token, err := auth.IssueToken("drv_1", "driver@example.com", "sedan", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.ParseToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.DriverID != "drv_1" {
		t.Fatalf("driver id: got %s", claims.DriverID)
	}
	if claims.Email != "driver@example.com" {
		t.Fatalf("email: got %s", claims.Email)
	}
	if claims.PackageSlug != "sedan" {
		t.Fatalf("package: got %s", claims.PackageSlug)
	}
}

func TestParseTokenRejectsInvalid(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	if _, err := auth.ParseToken(""); err == nil {
		t.Fatal("expected error for empty token")
	}
	if _, err := auth.ParseToken("not-a-jwt"); err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestNewDriverID(t *testing.T) {
	id := auth.NewDriverID()
	if len(id) < 5 || id[:4] != "drv_" {
		t.Fatalf("unexpected id: %s", id)
	}
}
