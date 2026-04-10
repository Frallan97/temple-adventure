import { useState } from "react";
import { useGame } from "../hooks/useGame";
import { Terminal } from "../components/Terminal";
import { InventoryPanel } from "../components/InventoryPanel";
import { StartPage } from "./StartPage";

export function GamePage() {
  const game = useGame();
  const [showInventory, setShowInventory] = useState(false);
  const [started, setStarted] = useState(false);

  const handleNewGame = async (storyId: string, storyName: string) => {
    game.clearGame();
    await game.startGame(storyId, storyName);
    setStarted(true);
  };

  const handleResume = async () => {
    await game.resumeGame();
    setStarted(true);
  };

  if (!started) {
    return (
      <StartPage
        onNewGame={handleNewGame}
        onResume={handleResume}
        hasExistingGame={!!game.gameId}
        savedStoryName={game.storyName}
      />
    );
  }

  return (
    <div className="h-[100dvh] flex">
      <div className="flex-1">
        <Terminal
          output={game.output}
          onCommand={game.sendCommand}
          onNavigateHistory={game.navigateHistory}
          isLoading={game.isLoading}
          gameOver={game.gameOver}
        />
      </div>
      <InventoryPanel
        items={game.inventory}
        roomName={game.roomName}
        turnNumber={game.turnNumber}
        isOpen={showInventory}
        onToggle={() => setShowInventory(!showInventory)}
      />
    </div>
  );
}
