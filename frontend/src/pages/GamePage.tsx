import { useState } from "react";
import { useGame } from "../hooks/useGame";
import { Terminal } from "../components/Terminal";
import { InventoryPanel } from "../components/InventoryPanel";
import { GameOverOverlay } from "../components/GameOverOverlay";
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

  const handleMainMenu = () => {
    setStarted(false);
  };

  const handleNewGameFromOverlay = () => {
    game.clearGame();
    setStarted(false);
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
    <div className="h-[100dvh] flex relative">
      <div className="flex-1">
        <Terminal
          output={game.output}
          onCommand={game.sendCommand}
          onNavigateHistory={game.navigateHistory}
          isLoading={game.isLoading}
          gameOver={game.gameOver}
          onMainMenu={handleMainMenu}
        />
      </div>
      <InventoryPanel
        items={game.inventory}
        roomName={game.roomName}
        turnNumber={game.turnNumber}
        isOpen={showInventory}
        onToggle={() => setShowInventory(!showInventory)}
      />
      {game.gameOver && (
        <GameOverOverlay
          status={game.gameStatus}
          onMainMenu={handleNewGameFromOverlay}
        />
      )}
    </div>
  );
}
