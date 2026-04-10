import { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { storyApi } from "../lib/api-client";
import type { StorySummary } from "../types/story";

interface StartPageProps {
  onNewGame: (storyId: string, storyName: string) => void;
  onResume: () => void;
  hasExistingGame: boolean;
  savedStoryName: string | null;
}

export function StartPage({
  onNewGame,
  onResume,
  hasExistingGame,
  savedStoryName,
}: StartPageProps) {
  const [stories, setStories] = useState<StorySummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    storyApi
      .list()
      .then((resp) => {
        setStories(resp.stories || []);
        setLoading(false);
      })
      .catch((err) => {
        setError(`Failed to load stories: ${err.message}`);
        setLoading(false);
      });
  }, []);

  return (
    <div className="flex flex-col items-center justify-center min-h-screen bg-gray-950 font-mono text-amber-400 px-6 py-12">
      <div className="text-center space-y-10 max-w-2xl w-full">
        <div className="space-y-3">
          <h1 className="text-3xl sm:text-5xl font-bold tracking-widest drop-shadow-lg">
            TEXT ADVENTURE
          </h1>
          <p className="text-gray-500 text-sm tracking-wide">
            Choose your adventure
          </p>
        </div>

        {hasExistingGame && (
          <div className="border border-amber-800/40 rounded-xl p-5 bg-amber-950/20 shadow-lg shadow-amber-900/10">
            <p className="text-gray-400 text-sm mb-4">
              You have a game in progress
              {savedStoryName && (
                <span className="text-amber-400 font-semibold">
                  {" "}
                  — {savedStoryName}
                </span>
              )}
            </p>
            <button
              onClick={onResume}
              className="bg-amber-900/30 border border-amber-700 text-amber-400 px-8 py-2.5 rounded-lg hover:bg-amber-900/50 transition-all hover:shadow-md hover:shadow-amber-900/20"
            >
              Resume Game
            </button>
          </div>
        )}

        <div className="space-y-4">
          <h2 className="text-sm uppercase tracking-widest text-gray-500 border-b border-gray-800 pb-3">
            Available Stories
          </h2>

          {loading && (
            <p className="text-gray-500 animate-pulse py-4">
              Loading stories...
            </p>
          )}

          {error && <p className="text-red-400 text-sm py-2">{error}</p>}

          {!loading && stories.length === 0 && (
            <p className="text-gray-600 text-sm py-4">
              No stories available yet.
            </p>
          )}

          <div className="grid gap-4">
            {stories.map((story) => (
              <button
                key={story.id}
                onClick={() => onNewGame(story.id, story.name)}
                className="text-left border border-gray-800 rounded-xl p-4 sm:p-5 bg-gray-900/30 hover:bg-gray-900/60 hover:border-amber-700/50 transition-all hover:shadow-lg hover:shadow-amber-900/10 active:bg-gray-900/60 group"
              >
                <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-2 gap-1">
                  <h3 className="text-amber-400 font-bold group-hover:text-amber-300 transition-colors">
                    {story.name}
                  </h3>
                  <span className="text-gray-600 text-xs">
                    by {story.author}
                  </span>
                </div>
                {story.description && (
                  <p className="text-gray-400 text-sm leading-relaxed">
                    {story.description}
                  </p>
                )}
              </button>
            ))}
          </div>
        </div>

        <div className="pt-4 space-y-3 border-t border-gray-800/50">
          <p className="text-gray-600 text-xs max-w-md mx-auto leading-relaxed">
            Type commands to explore. Try:{" "}
            <span className="text-gray-500">
              look, move north, take item, use item, inventory, hint, help
            </span>
          </p>

          <Link
            to="/editor"
            className="inline-block text-gray-600 text-xs hover:text-amber-400/60 transition-colors"
          >
            Story Editor &rarr;
          </Link>
        </div>
      </div>
    </div>
  );
}
