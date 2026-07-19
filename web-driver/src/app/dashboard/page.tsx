"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "../../context/AuthContext"
import { DriverHeader } from "../../components/DriverHeader"
import { fetchDriverDashboard } from "../../lib/dashboard"
import type { DriverDashboard } from "../../types"
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card"
import { ScrollArea } from "../../components/ui/scroll-area"
import { Skeleton } from "../../components/ui/skeleton"

function formatINR(paise?: number) {
  if (paise == null) return "—"
  return `₹${(paise / 100).toFixed(2)}`
}

function formatWhen(iso?: string) {
  if (!iso) return "—"
  try {
    return new Date(iso).toLocaleString("en-IN", {
      dateStyle: "medium",
      timeStyle: "short",
    })
  } catch {
    return iso
  }
}

export default function DriverDashboardPage() {
  const { isAuthenticated, isLoading } = useAuth()
  const router = useRouter()
  const [data, setData] = useState<DriverDashboard | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.replace("/login")
    }
  }, [isAuthenticated, isLoading, router])

  useEffect(() => {
    if (!isAuthenticated) return
    let cancelled = false
    ;(async () => {
      setLoading(true)
      setError(null)
      try {
        const dash = await fetchDriverDashboard()
        if (!cancelled) setData(dash)
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : "Failed to load")
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [isAuthenticated])

  if (isLoading || !isAuthenticated) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-zinc-950">
        <div className="text-emerald-200/80 text-sm">Checking session…</div>
      </main>
    )
  }

  return (
    <main className="min-h-screen bg-zinc-950 text-zinc-100">
      <DriverHeader />
      <div className="mx-auto max-w-5xl px-4 py-8 space-y-8">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">Dashboard</h2>
          <p className="text-sm text-zinc-400 mt-1">
            Trip history, bonus points, and customer reviews
          </p>
        </div>

        {error && (
          <p className="text-sm text-red-400 rounded-lg border border-red-900/50 bg-red-950/40 px-3 py-2">
            {error}
          </p>
        )}

        <div className="grid gap-4 sm:grid-cols-3">
          <SummaryCard
            title="Trips completed"
            value={loading ? null : String(data?.tripCount ?? 0)}
          />
          <SummaryCard
            title="Bonus points"
            value={loading ? null : String(data?.bonusPoints ?? 0)}
          />
          <SummaryCard
            title="Average rating"
            value={
              loading
                ? null
                : data && data.averageRating > 0
                  ? data.averageRating.toFixed(1)
                  : "—"
            }
          />
        </div>

        <div className="grid gap-6 lg:grid-cols-2">
          <Card className="border-zinc-800 bg-zinc-900/60 text-zinc-100">
            <CardHeader>
              <CardTitle className="text-base">Trip history</CardTitle>
            </CardHeader>
            <CardContent>
              {loading ? (
                <div className="space-y-2">
                  <Skeleton className="h-12 w-full bg-zinc-800" />
                  <Skeleton className="h-12 w-full bg-zinc-800" />
                </div>
              ) : !data?.recentTrips?.length ? (
                <p className="text-sm text-zinc-500">No completed trips yet.</p>
              ) : (
                <ScrollArea className="h-[320px] pr-3">
                  <ul className="space-y-3">
                    {data.recentTrips.map((t) => (
                      <li
                        key={t.id}
                        className="rounded-lg border border-zinc-800 bg-zinc-950/50 px-3 py-2.5"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <span className="text-sm font-medium">{formatINR(t.fare)}</span>
                          <span className="text-xs uppercase tracking-wide text-emerald-400/80">
                            {t.status}
                          </span>
                        </div>
                        <p className="text-xs text-zinc-500 mt-1 truncate">
                          {formatWhen(t.completedAt)} · {t.id.slice(0, 8)}…
                        </p>
                      </li>
                    ))}
                  </ul>
                </ScrollArea>
              )}
            </CardContent>
          </Card>

          <Card className="border-zinc-800 bg-zinc-900/60 text-zinc-100">
            <CardHeader>
              <CardTitle className="text-base">Customer reviews</CardTitle>
            </CardHeader>
            <CardContent>
              {loading ? (
                <div className="space-y-2">
                  <Skeleton className="h-12 w-full bg-zinc-800" />
                  <Skeleton className="h-12 w-full bg-zinc-800" />
                </div>
              ) : !data?.recentReviews?.length ? (
                <p className="text-sm text-zinc-500">No reviews yet.</p>
              ) : (
                <ScrollArea className="h-[320px] pr-3">
                  <ul className="space-y-3">
                    {data.recentReviews.map((r) => (
                      <li
                        key={r.id}
                        className="rounded-lg border border-zinc-800 bg-zinc-950/50 px-3 py-2.5"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <span className="text-amber-400 text-sm tracking-wider">
                            {"★".repeat(r.rating)}
                            <span className="text-zinc-600">{"★".repeat(5 - r.rating)}</span>
                          </span>
                          <span className="text-xs text-emerald-400/80">+{r.bonusPoints} pts</span>
                        </div>
                        {r.comment && (
                          <p className="text-sm text-zinc-300 mt-1.5">{r.comment}</p>
                        )}
                        <p className="text-xs text-zinc-500 mt-1">{formatWhen(r.createdAt)}</p>
                      </li>
                    ))}
                  </ul>
                </ScrollArea>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </main>
  )
}

function SummaryCard({ title, value }: { title: string; value: string | null }) {
  return (
    <Card className="border-zinc-800 bg-zinc-900/60 text-zinc-100">
      <CardHeader className="pb-2">
        <CardTitle className="text-xs font-medium uppercase tracking-wide text-zinc-400">
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {value == null ? (
          <Skeleton className="h-8 w-16 bg-zinc-800" />
        ) : (
          <p className="text-3xl font-semibold tabular-nums text-emerald-300">{value}</p>
        )}
      </CardContent>
    </Card>
  )
}
