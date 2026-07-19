package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"ride-sharing/shared/auth"
	"ride-sharing/shared/env"
)

func driverServiceHTTPBase() string {
	if v := os.Getenv("DRIVER_SERVICE_HTTP_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://driver-service:8085"
}

func handleDriverRegister(w http.ResponseWriter, r *http.Request) {
	proxyDriverHTTP(w, r, "/register")
}

func handleDriverLogin(w http.ResponseWriter, r *http.Request) {
	proxyDriverHTTP(w, r, "/login")
}

func handleDriverMe(w http.ResponseWriter, r *http.Request) {
	proxyDriverHTTP(w, r, "/me")
}

func proxyDriverHTTP(w http.ResponseWriter, r *http.Request, path string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	req, err := http.NewRequestWithContext(r.Context(), r.Method, driverServiceHTTPBase()+path, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("driver auth proxy error: %v", err)
		http.Error(w, "driver service unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func notifyDriverOfferResponse(driverID, tripID string, accepted bool) {
	payload, _ := json.Marshal(map[string]any{
		"driverId": driverID,
		"tripId":   tripID,
		"accepted": accepted,
	})
	req, err := http.NewRequest(http.MethodPost, driverServiceHTTPBase()+"/internal/offer-response", bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("offer-response notify failed: %v", err)
		return
	}
	resp.Body.Close()
}

func resolveDriverIdentity(r *http.Request) (driverID, packageSlug string, err error) {
	token := r.URL.Query().Get("token")
	if token == "" {
		if h := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(h), "bearer ") {
			token = strings.TrimSpace(h[7:])
		}
	}
	if token != "" {
		claims, err := auth.ParseToken(token)
		if err != nil {
			return "", "", err
		}
		slug := claims.PackageSlug
		if q := r.URL.Query().Get("packageSlug"); q != "" {
			slug = q
		}
		return claims.DriverID, slug, nil
	}

	// Legacy demo path (seeded / anonymous) — only when explicitly allowed.
	if env.GetBool("ALLOW_ANON_DRIVER_WS", false) {
		userID := r.URL.Query().Get("userID")
		packageSlug = r.URL.Query().Get("packageSlug")
		if userID != "" && packageSlug != "" {
			return userID, packageSlug, nil
		}
	}
	return "", "", errMissingDriverAuth
}

var errMissingDriverAuth = errString("missing driver token")

type errString string

func (e errString) Error() string { return string(e) }
