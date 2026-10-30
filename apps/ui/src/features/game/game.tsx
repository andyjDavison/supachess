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
import { PanelTabs } from "../../components/ui/panel-tabs";
import { ChartIcon, ListIcon, PlusIcon, UsersIcon } from "../../components/navbar/icons";

const API_URL = import.meta.env.VITE_API_URL;

const TABS = [
  { id: "Analysis", label: "Analysis", icon: <ChartIcon /> },
  { id: "New Game", label: "New Game", icon: <PlusIcon /> },
  { id: "Games", label: "Games", icon: <ListIcon /> },
  { id: "Players", label: "Players", icon: <UsersIcon /> },
] as const;
type Tab = (typeof TABS)[number]["id"];

export default function GamePage() {
  const { gameId } = useParams<{ gameId: string }>();
  const currentGame = useGameStore((s) => s.currentGame);
  const setInitialGame = useGameStore((s) => s.setInitialGame);
  const setInitialMoves = useGameStore((s) => s.setInitialMoves);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [selectedTab, setSelectedTab] = useState<Tab>("Analysis");

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
        <PanelTabs tabs={TABS} selected={selectedTab} onSelect={setSelectedTab} label="Game panel" />
        {/* min-h-0 lets this shrink below its content height, which is
            what allows the panel's own overflow-y-auto to engage instead
            of the list stretching the row. */}
        <div className="flex flex-col flex-1 min-h-0">
          {selectedTab === "Analysis" && <AnalysisSidePanel game={currentGame} />}
          {selectedTab === "New Game" && <NewGameSidePanel />}
          {selectedTab === "Games" && <GamesSidePanel />}
          {selectedTab === "Players" && <PlayersSidePanel />}
        </div>
      </div>
    </div>
  );
}
