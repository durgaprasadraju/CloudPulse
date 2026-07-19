"use client";

import { useState } from "react";
import { Button } from "./ui/button";
import { API_URL } from "../constants";
import { BackendEndpoints } from "../contracts";

interface TripFeedbackFormProps {
  tripID: string;
  userID: string;
  onSubmitted: () => void;
  onSkip: () => void;
}

export function TripFeedbackForm({ tripID, userID, onSubmitted, onSkip }: TripFeedbackFormProps) {
  const [rating, setRating] = useState(5);
  const [comment, setComment] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState(false);

  const handleSubmit = async () => {
    setSubmitting(true);
    setError(null);
    try {
      const res = await fetch(`${API_URL}${BackendEndpoints.TRIP_REVIEW(tripID)}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ userID, rating, comment: comment.trim() }),
      });
      if (!res.ok) {
        throw new Error((await res.text()) || "Failed to submit review");
      }
      setDone(true);
      onSubmitted();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to submit review");
    } finally {
      setSubmitting(false);
    }
  };

  if (done) {
    return (
      <div className="text-center space-y-3 py-2">
        <p className="text-lg font-semibold text-emerald-700">Thanks for your feedback!</p>
        <p className="text-sm text-gray-600">Your driver earned {rating} bonus point{rating === 1 ? "" : "s"}.</p>
        <Button variant="outline" className="w-full" onClick={onSkip}>
          Done
        </Button>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div>
        <p className="text-sm font-medium text-gray-700 mb-2">Rate your ride</p>
        <div className="flex gap-1">
          {[1, 2, 3, 4, 5].map((n) => (
            <button
              key={n}
              type="button"
              onClick={() => setRating(n)}
              className={`h-10 w-10 rounded-md text-lg transition ${
                n <= rating
                  ? "bg-amber-400 text-white"
                  : "bg-gray-100 text-gray-400 hover:bg-gray-200"
              }`}
              aria-label={`${n} star${n === 1 ? "" : "s"}`}
            >
              ★
            </button>
          ))}
        </div>
      </div>
      <div>
        <label className="text-xs font-medium text-gray-600">Comment (optional)</label>
        <textarea
          className="mt-1 w-full rounded-md border px-3 py-2 text-sm min-h-[80px]"
          value={comment}
          onChange={(e) => setComment(e.target.value)}
          placeholder="How was the ride?"
          maxLength={500}
        />
      </div>
      {error && <p className="text-xs text-red-600">{error}</p>}
      <Button onClick={handleSubmit} disabled={submitting} className="w-full">
        {submitting ? "Submitting…" : "Submit review"}
      </Button>
      <Button variant="ghost" className="w-full text-gray-500" onClick={onSkip} disabled={submitting}>
        Skip
      </Button>
    </div>
  );
}
