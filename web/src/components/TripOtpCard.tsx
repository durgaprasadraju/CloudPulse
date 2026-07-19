"use client";

import { useState } from "react";
import { Button } from "./ui/button";
import { Copy, Share2 } from "lucide-react";

interface TripOtpCardProps {
  otp: string;
  driverName?: string;
}

export function TripOtpCard({ otp, driverName }: TripOtpCardProps) {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(otp);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      /* ignore */
    }
  };

  const share = async () => {
    const text = `CloudPulse pickup OTP: ${otp}${driverName ? ` (share with ${driverName})` : ""}`;
    if (navigator.share) {
      try {
        await navigator.share({ title: "Pickup OTP", text });
        return;
      } catch {
        /* fall through */
      }
    }
    await copy();
  };

  return (
    <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-4 space-y-3">
      <div>
        <p className="text-xs font-medium uppercase tracking-wide text-emerald-800">
          Share this OTP with your driver
        </p>
        <p className="text-3xl font-bold tracking-[0.35em] text-emerald-900 mt-1 font-mono">
          {otp}
        </p>
        <p className="text-xs text-emerald-700 mt-2">
          The trip starts only after the driver enters this code at pickup.
        </p>
      </div>
      <div className="flex gap-2">
        <Button variant="outline" className="flex-1" onClick={copy}>
          <Copy className="size-4" />
          {copied ? "Copied" : "Copy"}
        </Button>
        <Button variant="outline" className="flex-1" onClick={share}>
          <Share2 className="size-4" />
          Share
        </Button>
      </div>
    </div>
  );
}
