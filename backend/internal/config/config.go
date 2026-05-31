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

package config

import (
	"fmt"
	"os"
)

// Config holds runtime configuration for the API server.
type Config struct {
	Host string
	Port string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	host := getenv("BEACON_HOST", "0.0.0.0")
	port := getenv("BEACON_PORT", "8080")
	return Config{Host: host, Port: port}
}

// Addr returns the listen address host:port.
func (c Config) Addr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
