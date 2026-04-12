import { useState, useEffect } from "react";
import { Link, useParams } from "react-router-dom";
import { gameApi } from "../lib/api-client";
import type { GameLogSummary, CommandEntry } from "../types/game";

function GameLogList() {
  const [games, setGames] = useState<GameLogSummary[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    try {
      const ids: string[] = JSON.parse(
        localStorage.getItem("temple_game_logs") || "[]"
      );
      if (ids.length === 0) {
        setLoading(false);
        return;
      }
      gameApi
        .getLogs(ids)
        .then((resp) => {
          setGames(resp.games);
          setLoading(false);
        })
        .catch(() => setLoading(false));
    } catch {
      setLoading(false);
    }
  }, []);

  const statusLabel = (status: string) => {
    if (status === "completed") return { text: "Won", color: "text-green-400" };
    if (status === "failed") return { text: "Failed", color: "text-red-400" };
    return { text: "In Progress", color: "text-yellow-400" };
  };

  return (
    <div className="flex flex-col items-center min-h-screen bg-gray-950 font-mono text-amber-400 px-6 py-12">
      <div className="max-w-2xl w-full space-y-8">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold tracking-widest">GAME LOGS</h1>
          <Link
            to="/"
            className="text-gray-500 text-sm hover:text-amber-400 transition-colors"
          >
            &larr; Back
          </Link>
        </div>

        {loading && (
          <p className="text-gray-500 animate-pulse">Loading...</p>
        )}

        {!loading && games.length === 0 && (
          <p className="text-gray-600 text-sm">
            No game logs yet. Complete a game to see it here.
          </p>
        )}

        <div className="space-y-3">
          {games.map((game) => {
            const s = statusLabel(game.status);
            return (
              <Link
                key={game.id}
                to={`/logs/${game.id}`}
                className="block border border-gray-800 rounded-xl p-4 bg-gray-900/30 hover:bg-gray-900/60 hover:border-amber-700/50 transition-all"
              >
                <div className="flex items-center justify-between mb-1">
                  <span className="text-amber-400 font-bold">
                    {game.story_name}
                  </span>
                  <span className={`text-xs font-semibold ${s.color}`}>
                    {s.text}
                  </span>
                </div>
                <div className="flex items-center gap-4 text-gray-500 text-xs">
                  <span>{game.turn_number} turns</span>
                  <span>
                    {new Date(game.created_at).toLocaleDateString()}
                  </span>
                </div>
              </Link>
            );
          })}
        </div>
      </div>
    </div>
  );
}

function GameLogDetail() {
  const { id } = useParams<{ id: string }>();
  const [commands, setCommands] = useState<CommandEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    gameApi
      .getHistory(id)
      .then((resp) => {
        setCommands(resp.commands || []);
        setLoading(false);
      })
      .catch((err) => {
        setError(`Failed to load: ${err.message}`);
        setLoading(false);
      });
  }, [id]);

  return (
    <div className="flex flex-col items-center min-h-screen bg-gray-950 font-mono text-amber-400 px-6 py-12">
      <div className="max-w-2xl w-full space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold tracking-widest">GAME LOG</h1>
          <Link
            to="/logs"
            className="text-gray-500 text-sm hover:text-amber-400 transition-colors"
          >
            &larr; All Logs
          </Link>
        </div>

        {loading && (
          <p className="text-gray-500 animate-pulse">Loading transcript...</p>
        )}

        {error && <p className="text-red-400 text-sm">{error}</p>}

        {!loading && commands.length === 0 && (
          <p className="text-gray-600 text-sm">No commands recorded.</p>
        )}

        <div className="space-y-1 border border-gray-800 rounded-xl p-4 sm:p-6 bg-gray-900/30 max-h-[70vh] overflow-y-auto">
          {commands.map((cmd) => (
            <div key={cmd.id}>
              <div className="text-green-400 text-sm">
                <span className="text-gray-600 text-xs mr-2">
                  [{cmd.turn_number}]
                </span>
                &gt; {cmd.raw_input}
              </div>
              <div className="text-gray-300 text-sm whitespace-pre-wrap mb-3 ml-6">
                {cmd.response_text}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

export { GameLogList, GameLogDetail };
