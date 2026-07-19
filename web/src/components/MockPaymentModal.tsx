"use client";

import { useState } from "react";
import { Button } from "./ui/button";
import { PaymentEventSessionCreatedData } from "../contracts";

interface MockPaymentModalProps {
  paymentSession: PaymentEventSessionCreatedData;
  driverName?: string;
  distanceMeters?: number;
  durationSeconds?: number;
  onPaid: () => void;
}

function formatAmount(amount: number, currency: string) {
  // Backend stores fare as totalPriceInCents in some paths, dollars in others.
  // Local mock sessions use the raw number from the fare model.
  const display = amount > 1000 ? (amount / 100).toFixed(2) : amount.toFixed(2);
  return `${currency} ${display}`;
}

export function MockPaymentModal({
  paymentSession,
  driverName,
  distanceMeters,
  durationSeconds,
  onPaid,
}: MockPaymentModalProps) {
  const [card, setCard] = useState("4242 4242 4242 4242");
  const [exp, setExp] = useState("12/28");
  const [cvc, setCvc] = useState("123");
  const [paying, setPaying] = useState(false);

  const handlePay = async () => {
    setPaying(true);
    await new Promise((r) => setTimeout(r, 900));
    setPaying(false);
    onPaid();
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="rounded-lg border bg-slate-50 p-3 text-sm space-y-1">
        <p className="font-semibold text-base">
          {formatAmount(paymentSession.amount, paymentSession.currency)}
        </p>
        {driverName && <p className="text-gray-600">Driver: {driverName}</p>}
        {distanceMeters != null && (
          <p className="text-gray-600">Distance: {(distanceMeters / 1000).toFixed(1)} km</p>
        )}
        {durationSeconds != null && (
          <p className="text-gray-600">Duration: {Math.round(durationSeconds / 60)} min</p>
        )}
        <p className="text-xs text-gray-400">Trip {paymentSession.tripID}</p>
      </div>

      <div className="space-y-2">
        <label className="text-xs font-medium text-gray-600">Card number</label>
        <input
          className="w-full rounded-md border px-3 py-2 text-sm"
          value={card}
          onChange={(e) => setCard(e.target.value)}
        />
        <div className="flex gap-2">
          <div className="flex-1">
            <label className="text-xs font-medium text-gray-600">Expiry</label>
            <input
              className="w-full rounded-md border px-3 py-2 text-sm"
              value={exp}
              onChange={(e) => setExp(e.target.value)}
            />
          </div>
          <div className="flex-1">
            <label className="text-xs font-medium text-gray-600">CVC</label>
            <input
              className="w-full rounded-md border px-3 py-2 text-sm"
              value={cvc}
              onChange={(e) => setCvc(e.target.value)}
            />
          </div>
        </div>
        <p className="text-xs text-gray-400">Demo checkout — no real charge is made.</p>
      </div>

      <Button onClick={handlePay} disabled={paying} className="w-full">
        {paying ? "Processing…" : `Pay ${formatAmount(paymentSession.amount, paymentSession.currency)}`}
      </Button>
    </div>
  );
}
