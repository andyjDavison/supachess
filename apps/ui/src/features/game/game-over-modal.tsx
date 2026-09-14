import { useNavigate } from "react-router";
import { useGameStore } from "./store";
import { useAuth } from "../../stores/AuthContext";
import type { Game } from "../../lib/ws/schemas";

interface GameOverModalProps {
  game: Game;
}

// Human-readable text for each result/reason combination. Kept as a
// lookup rather than string concatenation so each case reads naturally
// ("Checkmate" vs "won by checkmate" vs "White wins by checkmate") -
// concatenating a generic template tends to produce slightly awkward
// phrasing for at least one of the cases.
function describeOutcome(
  game: Game,
  isWhite: boolean,
): { headline: string; detail: string } {
  if (game.result === "draw") {
    const reasonText: Record<string, string> = {
      stalemate: "Stalemate",
      threefold_repetition: "Draw by repetition",
      fifty_move_rule: "Draw by the fifty-move rule",
      insufficient_material: "Draw by insufficient material",
      draw_agreement: "Draw by agreement",
    };
    return {
      headline: "Draw",
      detail: reasonText[game.resultReason ?? ""] ?? "The game ended in a draw",
    };
  }

  const won =
    (isWhite && game.result === "white_wins") ||
    (!isWhite && game.result === "black_wins");

  const reasonText: Record<string, string> = {
    checkmate: "by checkmate",
    resignation: "by resignation",
    timeout: "on time",
  };
  const reason = reasonText[game.resultReason ?? ""] ?? "";

  return {
    headline: won ? "You won!" : "You lost",
    detail: reason ? `${won ? "Won" : "Lost"} ${reason}` : "",
  };
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
      <div className="w-full max-w-sm rounded-lg bg-white p-6 text-center shadow-xl">
        <h2 id="game-over-title" className="text-2xl font-bold">
          {headline}
        </h2>
        {detail && <p className="mt-1 text-gray-500">{detail}</p>}

        <div className="mt-6 flex flex-col gap-2">
          {/* Analyze is a placeholder for now - closes the modal and
              leaves the user on the finished board, rather than
              navigating anywhere. A real analysis view is a separate,
              larger feature. */}
          <button
            className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium"
            onClick={() => {
              /* no-op for now - just lets the modal close via its own dismiss, see GamePage */
            }}
          >
            Analyze game
          </button>
          <button
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white"
            onClick={handleBackToLobby}
          >
            Back to lobby
          </button>
        </div>
      </div>
    </div>
  );
}
