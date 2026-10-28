import type { Game } from "../../lib/ws/schemas";

/** "1 min", "10 min", or "3 | 2" for games with an increment. */
export function formatTimeControl(initialMs: number, incrementMs: number): string {
  const minutes = Math.round(initialMs / 60_000);
  const increment = Math.round(incrementMs / 1000);
  return increment > 0 ? `${minutes} | ${increment}` : `${minutes} min`;
}

const DRAW_REASONS: Record<string, string> = {
  stalemate: "stalemate",
  threefold_repetition: "repetition",
  fifty_move_rule: "the fifty-move rule",
  insufficient_material: "insufficient material",
  draw_agreement: "agreement",
};

const WIN_REASONS: Record<string, string> = {
  checkmate: "by checkmate",
  resignation: "by resignation",
  timeout: "on time",
};

/**
 * The Game Over line in the game chat, e.g. "alice won by checkmate" or
 * "Game drawn by stalemate". Neutral, naming the winner, unlike
 * describeOutcome's "You won!" for the modal.
 */
export function describeGameEnd(game: Game, whiteName: string, blackName: string): string {
  if (game.result === "draw") {
    const reason = DRAW_REASONS[game.resultReason ?? ""];
    return reason ? `Game drawn by ${reason}` : "Game drawn";
  }
  const winner = game.result === "white_wins" ? whiteName : blackName;
  const reason = WIN_REASONS[game.resultReason ?? ""];
  return reason ? `${winner} won ${reason}` : `${winner} won`;
}
