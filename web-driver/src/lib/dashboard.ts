import { API_URL } from "../constants";
import { getStoredToken } from "./auth";
import type { DriverDashboard, DriverTripSummary } from "../types";

async function authFetch(path: string): Promise<Response> {
  const token = getStoredToken();
  if (!token) {
    throw new Error("Not signed in");
  }
  return fetch(`${API_URL}${path}`, {
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
  });
}

export async function fetchDriverDashboard(): Promise<DriverDashboard> {
  const res = await authFetch("/drivers/me/dashboard");
  if (!res.ok) {
    throw new Error((await res.text()) || "Failed to load dashboard");
  }
  return res.json();
}

export async function fetchDriverTrips(): Promise<DriverTripSummary[]> {
  const res = await authFetch("/drivers/me/trips");
  if (!res.ok) {
    throw new Error((await res.text()) || "Failed to load trips");
  }
  const data = (await res.json()) as { trips?: DriverTripSummary[] };
  return data.trips ?? [];
}
