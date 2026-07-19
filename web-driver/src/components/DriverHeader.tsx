"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { useAuth } from "../context/AuthContext"
import { Avatar, AvatarFallback, AvatarImage } from "./ui/avatar"
import { Button } from "./ui/button"
import { PackagesMeta } from "./PackagesMeta"
import { LogOut, LayoutDashboard, Map } from "lucide-react"

interface DriverHeaderProps {
  online?: boolean
  connected?: boolean
  onToggleOnline?: () => void
  showOnlineControls?: boolean
}

export function DriverHeader({
  online = false,
  connected = false,
  onToggleOnline,
  showOnlineControls = false,
}: DriverHeaderProps) {
  const { profile, logout } = useAuth()
  const pathname = usePathname()
  const initials = (profile?.name ?? "D")
    .split(" ")
    .map((p) => p[0])
    .join("")
    .slice(0, 2)
    .toUpperCase()

  const packageMeta = profile?.packageSlug ? PackagesMeta[profile.packageSlug] : null
  const onMap = pathname === "/"
  const onDashboard = pathname?.startsWith("/dashboard")

  return (
    <header className="flex items-center justify-between gap-4 bg-gradient-to-r from-zinc-950 via-emerald-950 to-zinc-950 px-4 py-3 text-white shadow-lg">
      <div className="flex items-center gap-3 min-w-0">
        <div className="hidden sm:flex h-9 w-9 items-center justify-center rounded-lg bg-emerald-600/30 ring-1 ring-emerald-500/40">
          <span className="text-sm font-bold tracking-tight">CP</span>
        </div>
        <div className="min-w-0">
          <h1 className="text-base sm:text-lg font-semibold tracking-tight truncate">
            CloudPulse Driver
          </h1>
          <p className="text-xs text-emerald-200/70 truncate">
            {profile?.name}
            {profile?.carPlate ? ` · ${profile.carPlate.toUpperCase()}` : ""}
            {packageMeta ? ` · ${packageMeta.name}` : ""}
          </p>
        </div>
      </div>

      <div className="flex items-center gap-2 sm:gap-3 shrink-0">
        <nav className="flex items-center rounded-lg bg-black/30 p-0.5 ring-1 ring-white/10">
          <Link
            href="/"
            className={`inline-flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-xs font-medium transition ${
              onMap ? "bg-emerald-600 text-white" : "text-zinc-300 hover:text-white"
            }`}
          >
            <Map className="size-3.5" />
            Map
          </Link>
          <Link
            href="/dashboard"
            className={`inline-flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-xs font-medium transition ${
              onDashboard ? "bg-emerald-600 text-white" : "text-zinc-300 hover:text-white"
            }`}
          >
            <LayoutDashboard className="size-3.5" />
            Dashboard
          </Link>
        </nav>

        {showOnlineControls && onToggleOnline && (
          <>
            <span
              className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ring-1 ${
                online
                  ? "bg-emerald-500/20 text-emerald-300 ring-emerald-400/40"
                  : "bg-zinc-800 text-zinc-400 ring-zinc-600/50"
              }`}
            >
              <span
                className={`h-1.5 w-1.5 rounded-full ${
                  online ? (connected ? "bg-emerald-400 animate-pulse" : "bg-amber-400") : "bg-zinc-500"
                }`}
              />
              {online ? (connected ? "Online" : "Connecting…") : "Offline"}
            </span>

            <Button
              size="sm"
              onClick={onToggleOnline}
              className={
                online
                  ? "bg-zinc-800 hover:bg-zinc-700 text-white border border-zinc-600"
                  : "bg-emerald-600 hover:bg-emerald-500 text-white"
              }
            >
              {online ? "Go Offline" : "Go Online"}
            </Button>
          </>
        )}

        <Avatar className="h-9 w-9 ring-2 ring-emerald-700/50">
          {profile?.profilePicture && (
            <AvatarImage src={profile.profilePicture} alt={profile.name} />
          )}
          <AvatarFallback className="bg-emerald-900 text-emerald-100 text-xs">
            {initials}
          </AvatarFallback>
        </Avatar>

        <Button
          size="icon"
          variant="ghost"
          className="text-zinc-300 hover:text-white hover:bg-white/10"
          onClick={logout}
          title="Sign out"
        >
          <LogOut className="size-4" />
        </Button>
      </div>
    </header>
  )
}
