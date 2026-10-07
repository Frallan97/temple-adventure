import { useRef, useEffect } from "react";
import type { CommandContext, OutputEntry } from "../types/game";
import { OutputLine } from "./OutputLine";
import { CommandInput } from "./CommandInput";
import { PixelButton } from "./PixelButton";

interface TerminalProps {
  output: OutputEntry[];
  commandContext?: CommandContext;
  onCommand: (input: string) => void;
  onNavigateHistory: (direction: "up" | "down") => string;
  isLoading: boolean;
  gameOver: boolean;
  onMainMenu: () => void;
  inventoryOpen: boolean;
  onToggleInventory: () => void;
}

export function Terminal({
  output,
  commandContext,
  onCommand,
  onNavigateHistory,
  isLoading,
  gameOver,
  onMainMenu,
  inventoryOpen,
  onToggleInventory,
}: TerminalProps) {
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [output]);

  return (
    <div className="flex flex-col h-[100dvh] bg-gray-950 font-mono text-sm">
      {/* A real header row rather than fixed-position buttons, so the output
          below can never scroll underneath it. Sits above the inventory panel
          so its buttons stay clickable while the panel is open. */}
      <header className="relative z-30 flex items-center justify-between gap-2 border-b-2 border-gray-800 bg-gray-950 px-3 py-2 sm:px-5">
        <PixelButton onClick={onMainMenu} title="Back to the main menu">
          Menu
        </PixelButton>

        <div className="flex items-center gap-2">
          <PixelButton
            tone="cyan"
            onClick={() => onCommand("help")}
            disabled={isLoading || gameOver}
            title="List the available commands"
          >
            Help
          </PixelButton>
          <PixelButton
            tone="amber"
            onClick={onToggleInventory}
            title={inventoryOpen ? "Close the inventory" : "Open the inventory"}
          >
            {inventoryOpen ? "Close" : "Items"}
          </PixelButton>
        </div>
      </header>

      <div
        ref={scrollRef}
        className="flex-1 overflow-y-auto space-y-3 px-3 sm:px-5 py-4 scrollbar-thin"
      >
        {output.map((entry, i) => (
          <OutputLine key={i} entry={entry} isLatest={i === output.length - 1} />
        ))}
        {isLoading && (
          <div className="text-gray-600 animate-pulse flex items-center gap-2">
            <span className="inline-block w-1.5 h-1.5 bg-amber-400/60 animate-bounce" />
            Processing...
          </div>
        )}
      </div>

      <div className="px-3 sm:px-5 pb-[env(safe-area-inset-bottom,8px)] pt-2">
        <CommandInput
          context={commandContext}
          onSubmit={onCommand}
          onNavigateHistory={onNavigateHistory}
          disabled={isLoading || gameOver}
        />
      </div>
    </div>
  );
}
