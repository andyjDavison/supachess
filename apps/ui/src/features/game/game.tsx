import { useEffect, useState } from "react";
import { useParams } from "react-router";
import { useGameStore } from "../game/store";
import { gameSchema } from "../../lib/ws/schemas";
import { GameBoard } from "./game-board";
import { GameOverModal } from "./game-over-modal";
import AnalysisSidePanel from "../play/side-panel/analysis";
import NewGameSidePanel from "../play/side-panel/new-game";
import GamesSidePanel from "../play/side-panel/games";
import PlayersSidePanel from "../play/side-panel/players";

const API_URL = import.meta.env.VITE_API_URL;

export default function GamePage() {
  const { gameId } = useParams<{ gameId: string }>();
  const currentGame = useGameStore((s) => s.currentGame);
  const setInitialGame = useGameStore((s) => s.setInitialGame);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [selectedTab, setSelectedTab] = useState<string>("");

  useEffect(() => {
    if (!gameId) return;

    let cancelled = false;

    fetch(`${API_URL}/api/games/${gameId}`, { credentials: "include" })
      .then((res) => {
        if (!res.ok) throw new Error("game not found");
        return res.json();
      })
      .then((data) => {
        const parsed = gameSchema.parse(data);
        if (!cancelled) setInitialGame(parsed);
      })
      .catch((err) => {
        if (!cancelled) setLoadError(err.message);
      });

    return () => {
      cancelled = true;
    };
  }, [gameId, setInitialGame]);

  if (loadError) {
    return (
      <div className="p-8 text-red-600">
        Couldn't load this game: {loadError}
      </div>
    );
  }

  if (!currentGame) {
    return <div className="p-8">Loading game…</div>;
  }

  return (
    <div className="flex items-center justify-center w-full h-full gap-7">
      {currentGame.status === "finished" && (
        <GameOverModal game={currentGame} />
      )}
      <GameBoard game={currentGame} />
      <div className="flex flex-col h-full flex-1 min-w-75 max-w-120">
        <div className="flex flex-row justify-center items-center bg-options-header rounded-t-sm h-17">
          <div
            onClick={() => setSelectedTab("Analysis")}
            className={`flex items-center justify-center w-1/3 h-full ${selectedTab === "Analysis" ? "bg-nav-bg text-text" : ""} hover:cursor-pointer hover:text-text rounded-tl-md`}
          >
            Analysis
          </div>
          <div
            onClick={() => setSelectedTab("New Game")}
            className={`flex items-center justify-center w-1/3 h-full ${selectedTab === "New Game" ? "bg-nav-bg text-text" : ""} hover:cursor-pointer hover:text-text rounded-tl-md`}
          >
            New Game
          </div>
          <div
            onClick={() => setSelectedTab("Games")}
            className={`flex items-center justify-center w-1/3 h-full ${selectedTab === "Games" ? "bg-nav-bg text-text" : ""} hover:cursor-pointer hover:text-text`}
          >
            Games
          </div>
          <div
            onClick={() => setSelectedTab("Players")}
            className={`flex items-center justify-center w-1/3 h-full ${selectedTab === "Players" ? "bg-nav-bg text-text" : ""} hover:cursor-pointer hover:text-text rounded-tr-md`}
          >
            Players
          </div>
        </div>
        {selectedTab === "Analysis" && <AnalysisSidePanel />}
        {selectedTab === "New Game" && <NewGameSidePanel />}
        {selectedTab === "Games" && <GamesSidePanel />}
        {selectedTab === "Players" && <PlayersSidePanel />}
      </div>
    </div>
  );
}
