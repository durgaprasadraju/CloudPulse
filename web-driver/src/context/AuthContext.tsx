"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { API_URL } from "../constants";
import { BackendEndpoints } from "../contracts";
import {
  clearAuth,
  getStoredProfile,
  getStoredToken,
  storeAuth,
} from "../lib/auth";
import {
  AuthResponse,
  DriverProfile,
  LoginDriverPayload,
  RegisterDriverPayload,
} from "../types";

interface AuthContextValue {
  token: string | null;
  profile: DriverProfile | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (payload: LoginDriverPayload) => Promise<void>;
  register: (payload: RegisterDriverPayload) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

async function parseAuthResponse(res: Response): Promise<AuthResponse> {
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    const message =
      (body as { error?: string; message?: string })?.error ||
      (body as { message?: string })?.message ||
      `Request failed (${res.status})`;
    throw new Error(message);
  }

  const token =
    (body as AuthResponse).token ||
    (body as { accessToken?: string }).accessToken;
  const driver =
    (body as AuthResponse).driver ||
    (body as { profile?: DriverProfile }).profile;

  if (!token || !driver) {
    throw new Error("Invalid auth response: missing token or driver profile");
  }

  return { token, driver };
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [token, setToken] = useState<string | null>(null);
  const [profile, setProfile] = useState<DriverProfile | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    setToken(getStoredToken());
    setProfile(getStoredProfile());
    setIsLoading(false);
  }, []);

  const persist = useCallback((nextToken: string, nextProfile: DriverProfile) => {
    storeAuth(nextToken, nextProfile);
    setToken(nextToken);
    setProfile(nextProfile);
  }, []);

  const login = useCallback(
    async (payload: LoginDriverPayload) => {
      const res = await fetch(`${API_URL}${BackendEndpoints.DRIVERS_LOGIN}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data = await parseAuthResponse(res);
      persist(data.token, data.driver);
    },
    [persist]
  );

  const register = useCallback(
    async (payload: RegisterDriverPayload) => {
      const res = await fetch(`${API_URL}${BackendEndpoints.DRIVERS_REGISTER}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data = await parseAuthResponse(res);
      persist(data.token, data.driver);
    },
    [persist]
  );

  const logout = useCallback(() => {
    clearAuth();
    setToken(null);
    setProfile(null);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      token,
      profile,
      isAuthenticated: Boolean(token),
      isLoading,
      login,
      register,
      logout,
    }),
    [token, profile, isLoading, login, register, logout]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return ctx;
}
