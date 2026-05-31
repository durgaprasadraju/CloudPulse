/**
 * Copyright 2026 Durga Prasad Raju Nadimpalli
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { useCallback, useEffect, useState } from "react";
import type { HealthResponse } from "./types/api";

const API_BASE = import.meta.env.VITE_API_URL ?? "";

export default function App() {
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchHealth = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${API_BASE}/api/v1/health`);
      if (!res.ok) {
        throw new Error(`API returned ${String(res.status)}`);
      }
      const data = (await res.json()) as HealthResponse;
      setHealth(data);
    } catch (err) {
      setHealth(null);
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchHealth();
  }, [fetchHealth]);

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      <header className="border-b border-slate-800 bg-slate-900/80 backdrop-blur">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-4">
          <div className="flex items-center gap-3">
            <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-beacon-500 text-lg font-bold text-white">
              B
            </span>
            <div>
              <h1 className="text-lg font-semibold tracking-tight">Beacon</h1>
              <p className="text-xs text-slate-400">Open-source observability</p>
            </div>
          </div>
          <span className="rounded-full border border-slate-700 px-3 py-1 text-xs text-slate-400">
            v0.1.0
          </span>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-6 py-16">
        <section className="text-center">
          <h2 className="text-4xl font-bold tracking-tight text-white sm:text-5xl">
            See everything.{" "}
            <span className="text-beacon-500">Miss nothing.</span>
          </h2>
          <p className="mx-auto mt-4 max-w-2xl text-lg text-slate-400">
            Beacon is your foundation for metrics, logs, and traces — built for
            teams who want clarity without vendor lock-in.
          </p>
        </section>

        <section className="mt-12 rounded-xl border border-slate-800 bg-slate-900 p-6 shadow-xl">
          <div className="flex items-center justify-between">
            <h3 className="font-medium text-slate-200">API health</h3>
            <button
              type="button"
              onClick={() => void fetchHealth()}
              className="rounded-md bg-beacon-600 px-3 py-1.5 text-sm font-medium text-white transition hover:bg-beacon-500 disabled:opacity-50"
              disabled={loading}
            >
              {loading ? "Checking…" : "Refresh"}
            </button>
          </div>

          <div className="mt-4 font-mono text-sm">
            {loading && (
              <p className="text-slate-500">Connecting to backend…</p>
            )}
            {error && (
              <p className="text-red-400">
                Backend unreachable: {error}. Start the API on port 8080 or run{" "}
                <code className="rounded bg-slate-800 px-1">make dev</code>.
              </p>
            )}
            {health && !error && (
              <pre className="overflow-x-auto rounded-lg bg-slate-950 p-4 text-emerald-400">
                {JSON.stringify(health, null, 2)}
              </pre>
            )}
          </div>
        </section>
      </main>
    </div>
  );
}
