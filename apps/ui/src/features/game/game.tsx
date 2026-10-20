import { useEffect, useState } from "react";
import { useParams } from "react-router";
import { useGameStore } from "../game/store";
import { gameSchema, movesResponseSchema } from "../../lib/ws/schemas";
import { GameBoard } from "./game-board";
import { GameOverModal } from "./game-over-modal";
import AnalysisSidePanel from "../play/side-panel/analysis";
import NewGameSidePanel from "../play/side-panel/new-game";
import GamesSidePanel from "../play/side-panel/games";
import PlayersSidePanel from "../play/side-panel/players";

const API_URL = import.meta.env.VITE_API_URL;

const TABS = ["Analysis", "New Game", "Games", "Players"] as const;

export default function GamePage() {
  const { gameId } = useParams<{ gameId: string }>();
  const currentGame = useGameStore((s) => s.currentGame);
  const setInitialGame = useGameStore((s) => s.setInitialGame);
  const setInitialMoves = useGameStore((s) => s.setInitialMoves);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [selectedTab, setSelectedTab] = useState<string>("Analysis");

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

  // Deliberately a separate effect from the game fetch above: the game is
  // load-critical and sets loadError, but failing to load move history
  // should leave an empty list rather than blank the whole page.
  useEffect(() => {
    if (!gameId) return;

    let cancelled = false;

    fetch(`${API_URL}/api/games/${gameId}/moves`, { credentials: "include" })
      .then((res) => {
        if (!res.ok) throw new Error("could not load move history");
        return res.json();
      })
      .then((data) => {
        const parsed = movesResponseSchema.parse(data);
        if (!cancelled) setInitialMoves(parsed);
      })
      .catch((err) => {
        console.error("loading move history", err);
      });

    return () => {
      cancelled = true;
    };
  }, [gameId, setInitialMoves]);

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
    <div className="game-layout">
      {currentGame.status === "finished" && (
        <GameOverModal game={currentGame} />
      )}
      <div className="board-column">
        <GameBoard game={currentGame} />
      </div>
      <div className="side-panel">
        <div className="flex flex-row justify-center items-center bg-options-header rounded-t-sm h-17">
          {TABS.map((tab, i) => (
            <button
              key={tab}
              type="button"
              onClick={() => setSelectedTab(tab)}
              aria-pressed={selectedTab === tab}
              className={[
                "flex items-center justify-center w-1/4 h-full hover:cursor-pointer hover:text-text",
                selectedTab === tab ? "bg-nav-bg text-text" : "",
                i === 0 ? "rounded-tl-md" : "",
                i === TABS.length - 1 ? "rounded-tr-md" : "",
              ].join(" ")}
            >
              {tab}
            </button>
          ))}
        </div>
        {/* min-h-0 lets this shrink below its content height, which is
            what allows the panel's own overflow-y-auto to engage instead
            of the list stretching the row. */}
        <div className="flex flex-col flex-1 min-h-0">
          {selectedTab === "Analysis" && <AnalysisSidePanel />}
          {selectedTab === "New Game" && <NewGameSidePanel />}
          {selectedTab === "Games" && <GamesSidePanel />}
          {selectedTab === "Players" && <PlayersSidePanel />}
        </div>
      </div>
    </div>
  );
}
