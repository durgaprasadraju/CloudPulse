// Package otp provides pickup-code generation and verification for trip starts.
package otp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"time"
)

const (
	DefaultLength     = 6
	DefaultTTL        = 30 * time.Minute
	DefaultMaxAttempts = 5
)

func pepper() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "cloudpulse-dev-jwt-secret-change-me"
	}
	return []byte(secret)
}

// Generate returns a numeric OTP of the given length (default 6).
func Generate(length int) (string, error) {
	if length <= 0 {
		length = DefaultLength
	}
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length)), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	format := "%0" + strconv.Itoa(length) + "d"
	return fmt.Sprintf(format, n.Int64()), nil
}

// Hash returns an HMAC-SHA256 hex digest of the OTP.
func Hash(code string) string {
	mac := hmac.New(sha256.New, pepper())
	_, _ = mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify compares a plaintext OTP against a stored hash.
func Verify(code, hash string) bool {
	if code == "" || hash == "" {
		return false
	}
	expected, err := hex.DecodeString(hash)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, pepper())
	_, _ = mac.Write([]byte(code))
	return hmac.Equal(mac.Sum(nil), expected)
}
