package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"ride-sharing/shared/auth"
)

func startDriverHTTPServer(addr string, accounts *auth.AccountStore, svc *Service) {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /register", func(w http.ResponseWriter, r *http.Request) {
		if accounts == nil {
			writeAuthError(w, http.StatusServiceUnavailable, "accounts unavailable")
			return
		}
		var body struct {
			Email       string `json:"email"`
			Password    string `json:"password"`
			Name        string `json:"name"`
			Phone       string `json:"phone"`
			PackageSlug string `json:"packageSlug"`
			CarPlate    string `json:"carPlate"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAuthError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		acc, err := accounts.Register(r.Context(), auth.RegisterInput{
			Email:       body.Email,
			Password:    body.Password,
			Name:        body.Name,
			Phone:       body.Phone,
			PackageSlug: body.PackageSlug,
			CarPlate:    body.CarPlate,
		})
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, auth.ErrEmailTaken) {
				status = http.StatusConflict
			}
			writeAuthError(w, status, err.Error())
			return
		}
		token, err := auth.IssueToken(acc.ID, acc.Email, acc.PackageSlug, 24*time.Hour)
		if err != nil {
			writeAuthError(w, http.StatusInternalServerError, "failed to issue token")
			return
		}
		writeAuthJSON(w, http.StatusCreated, map[string]any{
			"token":  token,
			"driver": acc.PublicProfile(),
		})
	})

	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		if accounts == nil {
			writeAuthError(w, http.StatusServiceUnavailable, "accounts unavailable")
			return
		}
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAuthError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		acc, err := accounts.Authenticate(r.Context(), body.Email, body.Password)
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, err.Error())
			return
		}
		token, err := auth.IssueToken(acc.ID, acc.Email, acc.PackageSlug, 24*time.Hour)
		if err != nil {
			writeAuthError(w, http.StatusInternalServerError, "failed to issue token")
			return
		}
		writeAuthJSON(w, http.StatusOK, map[string]any{
			"token":  token,
			"driver": acc.PublicProfile(),
		})
	})

	mux.HandleFunc("GET /me", func(w http.ResponseWriter, r *http.Request) {
		if accounts == nil {
			writeAuthError(w, http.StatusServiceUnavailable, "accounts unavailable")
			return
		}
		claims, err := bearerClaims(r)
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, err.Error())
			return
		}
		acc, err := accounts.GetByID(r.Context(), claims.DriverID)
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, "driver not found")
			return
		}
		writeAuthJSON(w, http.StatusOK, map[string]any{"driver": acc.PublicProfile()})
	})

	mux.HandleFunc("POST /internal/offer-response", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			DriverID string `json:"driverId"`
			TripID   string `json:"tripId"`
			Accepted bool   `json:"accepted"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAuthError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if body.DriverID == "" || svc == nil {
			writeAuthError(w, http.StatusBadRequest, "driverId required")
			return
		}
		svc.ClearPendingOffer(body.DriverID, body.TripID)
		svc.MarkBusy(body.DriverID, body.Accepted)
		writeAuthJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{Addr: addr, Handler: withCORS(mux)}
	go func() {
		log.Printf("Driver HTTP listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Driver HTTP stopped: %v", err)
		}
	}()
}

func bearerClaims(r *http.Request) (*auth.Claims, error) {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return auth.ParseToken(strings.TrimSpace(h[7:]))
	}
	if q := r.URL.Query().Get("token"); q != "" {
		return auth.ParseToken(q)
	}
	return nil, errors.New("missing authorization")
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeAuthJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeAuthError(w http.ResponseWriter, status int, msg string) {
	writeAuthJSON(w, status, map[string]string{"error": msg})
}
