// Copyright 2026 Durga Prasad Raju Nadimpalli
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/yourusername/beacon/backend/pkg/response"
)

// Health handles GET /health and GET /api/v1/health.
type Health struct{}

// NewHealth returns a Health handler.
func NewHealth() *Health {
	return &Health{}
}

type healthPayload struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}

// ServeHTTP implements http.Handler.
func (h *Health) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, healthPayload{
		Status:    "ok",
		Service:   "beacon-api",
		Timestamp: time.Now().UTC(),
	})
}

// Ready is a minimal readiness probe (extend with DB checks later).
func Ready(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}
