import { useNavigate } from "react-router";
import { useGameStore } from "./store";
import { useAuth } from "../../stores/AuthContext";
import type { Game } from "../../lib/ws/schemas";
import { describeOutcome } from "./describe-outcome";

interface GameOverModalProps {
  game: Game;
}

export function GameOverModal({ game }: GameOverModalProps) {
  const navigate = useNavigate();
  const { user } = useAuth();
  const clearMatch = useGameStore((s) => s.clearMatch);

  const isWhite = user?.id === game.whiteId;
  const { headline, detail } = describeOutcome(game, isWhite);

  function handleBackToLobby() {
    clearMatch(); // defensive - clears any stale matchmaking state before leaving
    navigate("/play/online");
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="game-over-title"
    >
      <div className="w-full max-w-sm rounded-lg border border-input-border bg-nav-bg p-6 text-center text-text shadow-xl">
        <h2 id="game-over-title" className="text-2xl font-bold">
          {headline}
        </h2>
        {detail && <p className="mt-1 text-clock-black-d">{detail}</p>}

        <div className="mt-6 flex flex-col gap-2">
          {/* Analyze is a placeholder for now - closes the modal and
              leaves the user on the finished board, rather than
              navigating anywhere. A real analysis view is a separate,
              larger feature. */}
          <button
            className="flex items-center justify-center h-9 rounded-md text-text hover:cursor-pointer bg-linear-to-b from-accent to-accent-strong hover:bg-linear-to-b hover:from-accent-hover hover:to-accent-strong"
            onClick={() => {}}
          >
            Analyze game
          </button>
          <button
            className="flex items-center justify-center h-9 rounded-md text-text hover:cursor-pointer bg-linear-to-b from-accent to-accent-strong hover:bg-linear-to-b hover:from-accent-hover hover:to-accent-strong"
            onClick={handleBackToLobby}
          >
            Back to lobby
          </button>
        </div>
      </div>
    </div>
  );
}
