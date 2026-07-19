package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"ride-sharing/services/trip-service/internal/domain"
)

type Handler struct {
	svc domain.TripService
}

func NewHandler(svc domain.TripService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Mount(mux *http.ServeMux) {
	mux.HandleFunc("POST /trips/{id}/review", h.handleSubmitReview)
	mux.HandleFunc("GET /drivers/{id}/dashboard", h.handleDriverDashboard)
	mux.HandleFunc("GET /drivers/{id}/trips", h.handleDriverTrips)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

type reviewRequest struct {
	UserID  string `json:"userID"`
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

func (h *Handler) handleSubmitReview(w http.ResponseWriter, r *http.Request) {
	tripID := r.PathValue("id")
	var req reviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	review, err := h.svc.SubmitReview(r.Context(), tripID, req.UserID, req.Rating, strings.TrimSpace(req.Comment))
	if err != nil {
		writeReviewError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, review)
}

func (h *Handler) handleDriverDashboard(w http.ResponseWriter, r *http.Request) {
	driverID := r.PathValue("id")
	dash, err := h.svc.DriverDashboard(r.Context(), driverID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, serializeDashboard(dash))
}

func (h *Handler) handleDriverTrips(w http.ResponseWriter, r *http.Request) {
	driverID := r.PathValue("id")
	trips, err := h.svc.DriverTrips(r.Context(), driverID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]map[string]any, 0, len(trips))
	for _, t := range trips {
		out = append(out, serializeTrip(t))
	}
	writeJSON(w, http.StatusOK, map[string]any{"trips": out})
}

func writeReviewError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidRating):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, domain.ErrReviewExists):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, domain.ErrTripNotPaid), errors.Is(err, domain.ErrNotTripOwner), errors.Is(err, domain.ErrNoDriver):
		http.Error(w, err.Error(), http.StatusForbidden)
	default:
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func serializeDashboard(d *domain.DriverDashboard) map[string]any {
	trips := make([]map[string]any, 0, len(d.RecentTrips))
	for _, t := range d.RecentTrips {
		trips = append(trips, serializeTrip(t))
	}
	reviews := make([]map[string]any, 0, len(d.RecentReviews))
	for _, rev := range d.RecentReviews {
		reviews = append(reviews, map[string]any{
			"id":          rev.ID.Hex(),
			"tripID":      rev.TripID,
			"userID":      rev.UserID,
			"driverID":    rev.DriverID,
			"rating":      rev.Rating,
			"comment":     rev.Comment,
			"bonusPoints": rev.BonusPoints,
			"createdAt":   rev.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return map[string]any{
		"tripCount":     d.TripCount,
		"bonusPoints":   d.BonusPoints,
		"averageRating": d.AverageRating,
		"recentTrips":   trips,
		"recentReviews": reviews,
	}
}

func serializeTrip(t *domain.TripModel) map[string]any {
	m := map[string]any{
		"id":     t.ID.Hex(),
		"userID": t.UserID,
		"status": t.Status,
	}
	if t.RideFare != nil {
		m["fare"] = t.RideFare.TotalPriceInCents
		m["packageSlug"] = t.RideFare.PackageSlug
		m["currency"] = "INR"
	}
	if t.Driver != nil {
		m["driver"] = map[string]any{
			"id":   t.Driver.Id,
			"name": t.Driver.Name,
		}
	}
	if t.CompletedAt != nil {
		m["completedAt"] = t.CompletedAt.UTC().Format(time.RFC3339)
	}
	return m
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
