import type { Game } from "../../lib/ws/schemas";

/**
 * Turns a finished game into the headline and detail the game-over modal
 * shows, from the perspective of the player viewing it.
 *
 * Lives apart from game-over-modal.tsx so it is importable without
 * tripping react-refresh's components-only rule for that file.
 */
export function describeOutcome(
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
