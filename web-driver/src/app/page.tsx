"use client"

import "leaflet/dist/leaflet.css"
import icon from "leaflet/dist/images/marker-icon.png"
import iconShadow from "leaflet/dist/images/marker-shadow.png"
import dynamic from "next/dynamic"
import { useEffect } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "../context/AuthContext"

const DriverMap = dynamic(
  () => import("../components/DriverMap").then((mod) => mod.DriverMap),
  {
    ssr: false,
    loading: () => (
      <div className="flex h-screen items-center justify-center bg-zinc-950 text-emerald-200">
        Loading map…
      </div>
    ),
  }
)

if (typeof window !== "undefined") {
  import("leaflet").then((L) => {
    const DefaultIcon = L.default.icon({
      iconUrl: icon.src,
      shadowUrl: iconShadow.src,
      iconSize: [25, 41],
      iconAnchor: [12, 41],
    })
    L.default.Marker.prototype.options.icon = DefaultIcon
  })
}

export default function DashboardPage() {
  const { isAuthenticated, isLoading } = useAuth()
  const router = useRouter()

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.replace("/login")
    }
  }, [isAuthenticated, isLoading, router])

  if (isLoading || !isAuthenticated) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-zinc-950">
        <div className="text-emerald-200/80 text-sm">Checking session…</div>
      </main>
    )
  }

  return (
    <main className="min-h-screen">
      <DriverMap />
    </main>
  )
}
