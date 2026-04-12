import { useState, useCallback, useRef } from "react";
import type { ItemInfo, OutputEntry } from "../types/game";
import { gameApi } from "../lib/api-client";

export function useGame() {
  const [gameId, setGameId] = useState<string | null>(() => {
    return localStorage.getItem("temple_game_id");
  });
  const [storyId, setStoryId] = useState<string | null>(() => {
    return localStorage.getItem("temple_story_id");
  });
  const [storyName, setStoryName] = useState<string | null>(() => {
    return localStorage.getItem("temple_story_name");
  });
  const [output, setOutput] = useState<OutputEntry[]>([]);
  const [inventory, setInventory] = useState<ItemInfo[]>([]);
  const [roomName, setRoomName] = useState("");
  const [turnNumber, setTurnNumber] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [gameOver, setGameOver] = useState(false);
  const [gameStatus, setGameStatus] = useState("active");
  const [endingId, setEndingId] = useState<string | null>(null);
  const [endingTitle, setEndingTitle] = useState<string | null>(null);
  const [commandHistory, setCommandHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const loadingRef = useRef(false);
  const gameOverRef = useRef(false);

  const addOutput = useCallback((entry: OutputEntry) => {
    setOutput((prev) => [...prev, entry]);
  }, []);

  const startGame = useCallback(
    async (storyId: string, name: string) => {
      setIsLoading(true);
      try {
        const resp = await gameApi.create(storyId);
        setGameId(resp.id);
        setStoryId(storyId);
        setStoryName(name);
        localStorage.setItem("temple_game_id", resp.id);
        localStorage.setItem("temple_story_id", storyId);
        localStorage.setItem("temple_story_name", name);
        setRoomName(resp.room_name);
        setTurnNumber(resp.turn_number);
        setInventory(resp.inventory || []);
        setGameOver(false);
        setGameStatus("active");
        setEndingId(null);
        setEndingTitle(null);
        setOutput([
          {
            type: "system",
            text: `=== ${name.toUpperCase()} ===\n`,
          },
          { type: "narrative", text: resp.description },
        ]);
      } catch (err) {
        addOutput({
          type: "error",
          text: `Failed to start game: ${err}`,
        });
      } finally {
        setIsLoading(false);
      }
    },
    [addOutput]
  );

  const resumeGame = useCallback(async () => {
    if (!gameId) return;
    setIsLoading(true);
    try {
      const resp = await gameApi.getState(gameId);
      setRoomName(resp.room_name);
      setTurnNumber(resp.turn_number);
      setInventory(resp.inventory || []);
      setGameOver(resp.status !== "active");
      setGameStatus(resp.status);
      setOutput([
        { type: "system", text: "=== Game Resumed ===\n" },
        { type: "narrative", text: resp.description },
      ]);
    } catch {
      localStorage.removeItem("temple_game_id");
      localStorage.removeItem("temple_story_id");
      localStorage.removeItem("temple_story_name");
      setGameId(null);
      setStoryId(null);
      setStoryName(null);
    } finally {
      setIsLoading(false);
    }
  }, [gameId]);

  const sendCommand = useCallback(
    async (input: string) => {
      if (!gameId || loadingRef.current || gameOverRef.current) return;

      const trimmed = input.trim();
      if (!trimmed) return;

      setCommandHistory((prev) => [...prev, trimmed]);
      setHistoryIndex(-1);
      addOutput({ type: "command", text: `> ${trimmed}` });
      loadingRef.current = true;
      setIsLoading(true);

      try {
        const resp = await gameApi.sendCommand(gameId, trimmed);
        addOutput({ type: "narrative", text: resp.text });
        if (resp.choices && resp.choices.length > 0) {
          const choiceText = resp.choices
            .map((c) => `  ${c.index}. ${c.text}`)
            .join("\n");
          addOutput({ type: "system", text: choiceText });
        }
        setRoomName(resp.room_name);
        setTurnNumber(resp.turn_number);
        setInventory(resp.inventory || []);
        if (resp.game_over) {
          gameOverRef.current = true;
          setGameOver(true);
          setGameStatus(resp.game_status);
          setEndingId(resp.ending_id || null);
          setEndingTitle(resp.ending_title || null);
          // Save to game log history
          if (gameId) {
            try {
              const logs: string[] = JSON.parse(localStorage.getItem("temple_game_logs") || "[]");
              if (!logs.includes(gameId)) {
                logs.unshift(gameId);
                localStorage.setItem("temple_game_logs", JSON.stringify(logs.slice(0, 50)));
              }
            } catch { /* ignore */ }
          }
        }
      } catch (err) {
        addOutput({ type: "error", text: `Error: ${err}` });
      } finally {
        loadingRef.current = false;
        setIsLoading(false);
      }
    },
    [gameId, addOutput]
  );

  const navigateHistory = useCallback(
    (direction: "up" | "down"): string => {
      if (commandHistory.length === 0) return "";

      let newIndex: number;
      if (direction === "up") {
        newIndex =
          historyIndex === -1
            ? commandHistory.length - 1
            : Math.max(0, historyIndex - 1);
      } else {
        newIndex =
          historyIndex === -1
            ? -1
            : Math.min(commandHistory.length - 1, historyIndex + 1);
      }

      setHistoryIndex(newIndex);
      return newIndex >= 0 ? commandHistory[newIndex] : "";
    },
    [commandHistory, historyIndex]
  );

  const clearGame = useCallback(() => {
    localStorage.removeItem("temple_game_id");
    localStorage.removeItem("temple_story_id");
    localStorage.removeItem("temple_story_name");
    setGameId(null);
    setStoryId(null);
    setStoryName(null);
    setOutput([]);
    setInventory([]);
    setRoomName("");
    setTurnNumber(0);
    loadingRef.current = false;
    gameOverRef.current = false;
    setGameOver(false);
    setGameStatus("active");
    setEndingId(null);
    setEndingTitle(null);
    setCommandHistory([]);
    setHistoryIndex(-1);
  }, []);

  return {
    gameId,
    storyId,
    storyName,
    output,
    inventory,
    roomName,
    turnNumber,
    isLoading,
    gameOver,
    gameStatus,
    endingId,
    endingTitle,
    startGame,
    resumeGame,
    sendCommand,
    navigateHistory,
    clearGame,
  };
}
