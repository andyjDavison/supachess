import { useEffect, useState } from "react";
import { useParams } from "react-router";
import { useGameStore } from "../game/store";
import { gameSchema } from "../../lib/ws/schemas";
import { GameBoard } from "./game-board";
import { GameOverModal } from "./game-over-modal";

const API_URL = import.meta.env.VITE_API_URL;

export default function GamePage() {
  const { gameId } = useParams<{ gameId: string }>();
  const currentGame = useGameStore((s) => s.currentGame);
  const setInitialGame = useGameStore((s) => s.setInitialGame); // added below
  const [loadError, setLoadError] = useState<string | null>(null);

  // Initial load / refresh: no WS message has necessarily arrived yet,
  // so fetch current state directly rather than relying solely on
  // currentGame, which only gets populated by move_made/game_over.
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
    <div>
      {currentGame.status === "finished" && (
        <GameOverModal game={currentGame} />
      )}
      <GameBoard game={currentGame} />
    </div>
  );
}
