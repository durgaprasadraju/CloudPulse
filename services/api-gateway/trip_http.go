package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"ride-sharing/shared/auth"
)

func tripServiceHTTPBase() string {
	if v := os.Getenv("TRIP_SERVICE_HTTP_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://trip-service:8086"
}

func handleTripReview(w http.ResponseWriter, r *http.Request) {
	tripID := r.PathValue("id")
	proxyTripHTTP(w, r, http.MethodPost, "/trips/"+tripID+"/review")
}

func handleDriverDashboard(w http.ResponseWriter, r *http.Request) {
	driverID, err := requireDriverJWT(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	proxyTripHTTP(w, r, http.MethodGet, "/drivers/"+driverID+"/dashboard")
}

func handleDriverTrips(w http.ResponseWriter, r *http.Request) {
	driverID, err := requireDriverJWT(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	proxyTripHTTP(w, r, http.MethodGet, "/drivers/"+driverID+"/trips")
}

func requireDriverJWT(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return "", errMissingDriverAuth
	}
	claims, err := auth.ParseToken(strings.TrimSpace(h[7:]))
	if err != nil {
		return "", err
	}
	return claims.DriverID, nil
}

func proxyTripHTTP(w http.ResponseWriter, r *http.Request, method, path string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	req, err := http.NewRequestWithContext(r.Context(), method, tripServiceHTTPBase()+path, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("trip HTTP proxy error: %v", err)
		http.Error(w, "trip service unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
