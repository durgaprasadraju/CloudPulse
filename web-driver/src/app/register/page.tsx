"use client"

import Link from "next/link"
import { FormEvent, useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "../../context/AuthContext"
import { Button } from "../../components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../../components/ui/card"
import { CarPackageSlug } from "../../types"
import { PackagesMeta } from "../../components/PackagesMeta"

const PACKAGE_OPTIONS = Object.values(CarPackageSlug)

export default function RegisterPage() {
  const { register, isAuthenticated, isLoading } = useAuth()
  const router = useRouter()
  const [name, setName] = useState("")
  const [email, setEmail] = useState("")
  const [phone, setPhone] = useState("")
  const [password, setPassword] = useState("")
  const [packageSlug, setPackageSlug] = useState<CarPackageSlug>(CarPackageSlug.SEDAN)
  const [carPlate, setCarPlate] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      router.replace("/")
    }
  }, [isAuthenticated, isLoading, router])

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      await register({ name, email, phone, password, packageSlug, carPlate })
      router.push("/")
    } catch (err) {
      setError(err instanceof Error ? err.message : "Registration failed")
    } finally {
      setSubmitting(false)
    }
  }

  const inputClass =
    "w-full rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-white outline-none focus:ring-2 focus:ring-emerald-600"

  return (
    <main className="min-h-screen flex items-center justify-center bg-gradient-to-br from-zinc-950 via-emerald-950 to-zinc-900 px-4 py-10">
      <Card className="w-full max-w-lg border-emerald-900/40 bg-zinc-950/80 text-white shadow-2xl backdrop-blur">
        <CardHeader className="space-y-1">
          <p className="text-xs uppercase tracking-[0.2em] text-emerald-400/80">CloudPulse</p>
          <CardTitle className="text-2xl text-white">Driver registration</CardTitle>
          <CardDescription className="text-zinc-400">
            Create your driver account to start accepting trips.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="space-y-4">
            <div className="grid sm:grid-cols-2 gap-4">
              <div className="space-y-2">
                <label className="text-sm text-zinc-300" htmlFor="name">
                  Full name
                </label>
                <input
                  id="name"
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className={inputClass}
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm text-zinc-300" htmlFor="phone">
                  Phone
                </label>
                <input
                  id="phone"
                  type="tel"
                  required
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                  className={inputClass}
                />
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm text-zinc-300" htmlFor="email">
                Email
              </label>
              <input
                id="email"
                type="email"
                required
                autoComplete="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className={inputClass}
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm text-zinc-300" htmlFor="password">
                Password
              </label>
              <input
                id="password"
                type="password"
                required
                minLength={6}
                autoComplete="new-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className={inputClass}
              />
            </div>

            <div className="grid sm:grid-cols-2 gap-4">
              <div className="space-y-2">
                <label className="text-sm text-zinc-300" htmlFor="packageSlug">
                  Vehicle package
                </label>
                <select
                  id="packageSlug"
                  value={packageSlug}
                  onChange={(e) => setPackageSlug(e.target.value as CarPackageSlug)}
                  className={inputClass}
                >
                  {PACKAGE_OPTIONS.map((slug) => (
                    <option key={slug} value={slug}>
                      {PackagesMeta[slug].name}
                    </option>
                  ))}
                </select>
              </div>
              <div className="space-y-2">
                <label className="text-sm text-zinc-300" htmlFor="carPlate">
                  License plate
                </label>
                <input
                  id="carPlate"
                  required
                  value={carPlate}
                  onChange={(e) => setCarPlate(e.target.value)}
                  placeholder="TS09 AB 1234"
                  className={inputClass}
                />
              </div>
            </div>

            {error && (
              <p className="text-sm text-red-400 bg-red-950/40 border border-red-900/50 rounded-md px-3 py-2">
                {error}
              </p>
            )}

            <Button
              type="submit"
              disabled={submitting}
              className="w-full bg-emerald-600 hover:bg-emerald-500 text-white"
            >
              {submitting ? "Creating account…" : "Create account"}
            </Button>
          </form>

          <p className="mt-6 text-center text-sm text-zinc-400">
            Already registered?{" "}
            <Link href="/login" className="text-emerald-400 hover:underline">
              Sign in
            </Link>
          </p>
        </CardContent>
      </Card>
    </main>
  )
}
