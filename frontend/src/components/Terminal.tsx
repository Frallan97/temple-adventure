import { useRef, useEffect } from "react";
import type { OutputEntry } from "../types/game";
import { OutputLine } from "./OutputLine";
import { CommandInput } from "./CommandInput";

interface TerminalProps {
  output: OutputEntry[];
  onCommand: (input: string) => void;
  onNavigateHistory: (direction: "up" | "down") => string;
  isLoading: boolean;
  gameOver: boolean;
  onMainMenu: () => void;
}

export function Terminal({
  output,
  onCommand,
  onNavigateHistory,
  isLoading,
  gameOver,
  onMainMenu,
}: TerminalProps) {
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [output]);

  return (
    <div className="flex flex-col h-[100dvh] bg-gray-950 font-mono text-sm">
      <button
        onClick={onMainMenu}
        className="fixed top-3 left-3 sm:top-4 sm:left-4 z-10 bg-gray-900/90 backdrop-blur text-gray-400 px-3 sm:px-4 py-1.5 rounded-lg border border-gray-700/80 hover:bg-gray-800 hover:border-gray-600 hover:text-gray-300 text-sm transition-all shadow-lg active:bg-gray-800"
      >
        Menu
      </button>
      <div
        ref={scrollRef}
        className="flex-1 overflow-y-auto space-y-3 px-3 sm:px-5 py-4 scrollbar-thin"
      >
        {output.map((entry, i) => (
          <OutputLine
            key={i}
            entry={entry}
            isLatest={i === output.length - 1}
          />
        ))}
        {isLoading && (
          <div className="text-gray-600 animate-pulse flex items-center gap-2">
            <span className="inline-block w-1.5 h-1.5 bg-amber-400/60 rounded-full animate-bounce" />
            Processing...
          </div>
        )}
      </div>
      <div className="px-3 sm:px-5 pb-[env(safe-area-inset-bottom,8px)] pt-2">
        <CommandInput
          onSubmit={onCommand}
          onNavigateHistory={onNavigateHistory}
          disabled={isLoading || gameOver}
        />
      </div>
    </div>
  );
}
