import { useState } from "react";
import { StarRating } from "./StarRating";
import { storyApi } from "../lib/api-client";

interface GameOverOverlayProps {
  status: string;
  endingTitle?: string | null;
  storyId: string | null;
  gameId: string | null;
  onMainMenu: () => void;
}

export function GameOverOverlay({ status, endingTitle, storyId, gameId, onMainMenu }: GameOverOverlayProps) {
  const won = status === "completed";
  const [userRating, setUserRating] = useState<number>(0);
  const [submitted, setSubmitted] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const handleRate = async (rating: number) => {
    if (!storyId || !gameId || submitting) return;
    setUserRating(rating);
    setSubmitting(true);
    try {
      await storyApi.rate(storyId, gameId, rating);
      setSubmitted(true);
    } catch {
      // silently fail — rating is non-critical
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
      <div className="bg-gray-900 border border-gray-700 rounded-xl shadow-2xl px-8 py-10 max-w-sm w-full mx-4 text-center font-mono">
        {won ? (
          <>
            <div className="text-5xl mb-4">&#9734;</div>
            <h2 className="text-amber-400 text-2xl font-bold mb-2">
              {endingTitle || "Victory!"}
            </h2>
            <p className="text-gray-400 mb-6">
              You have conquered the temple and claimed your prize.
              Congratulations, adventurer!
            </p>
          </>
        ) : (
          <>
            <div className="text-5xl mb-4">&#9760;</div>
            <h2 className="text-red-400 text-2xl font-bold mb-2">
              Game Over
            </h2>
            <p className="text-gray-400 mb-6">
              The temple has claimed another soul. Perhaps next time you will
              fare better.
            </p>
          </>
        )}

        {storyId && gameId && (
          <div className="mb-6 space-y-2">
            <p className="text-gray-500 text-sm">
              {submitted ? "Thanks for rating!" : "Rate this adventure"}
            </p>
            <div className="flex justify-center">
              <StarRating
                rating={userRating}
                size="lg"
                interactive={!submitted}
                onRate={handleRate}
              />
            </div>
          </div>
        )}

        <button
          onClick={onMainMenu}
          className="w-full bg-amber-600 hover:bg-amber-500 text-gray-950 font-bold py-3 rounded-lg transition-colors"
        >
          Return to Main Menu
        </button>
      </div>
    </div>
  );
}
