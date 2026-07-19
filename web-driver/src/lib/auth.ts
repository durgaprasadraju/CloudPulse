import { DriverProfile } from "../types";

export const DRIVER_TOKEN_KEY = "cloudpulse_driver_token";
export const DRIVER_PROFILE_KEY = "cloudpulse_driver_profile";

export function getStoredToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(DRIVER_TOKEN_KEY);
}

export function getStoredProfile(): DriverProfile | null {
  if (typeof window === "undefined") return null;
  const raw = localStorage.getItem(DRIVER_PROFILE_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as DriverProfile;
  } catch {
    return null;
  }
}

export function storeAuth(token: string, profile: DriverProfile): void {
  localStorage.setItem(DRIVER_TOKEN_KEY, token);
  localStorage.setItem(DRIVER_PROFILE_KEY, JSON.stringify(profile));
}

export function clearAuth(): void {
  localStorage.removeItem(DRIVER_TOKEN_KEY);
  localStorage.removeItem(DRIVER_PROFILE_KEY);
}
