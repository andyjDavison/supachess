import type { Game, RatingCategory, SideRating } from "../../lib/ws/schemas";

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

/** "+8", "+0", "-3". */
function signed(n: number): string {
  return n >= 0 ? `+${n}` : `${n}`;
}

/** The stakes under New Game: "win +8 / draw +0 / lose -8". */
export function formatStakes(side: SideRating): string {
  return `win ${signed(side.win)} / draw ${signed(side.draw)} / lose ${signed(side.loss)}`;
}

const CATEGORY_NAMES: Record<RatingCategory, string> = { bullet: "Bullet", blitz: "Blitz", rapid: "Rapid" };

/** Under Game Over: "Your new Bullet rating is 110 (+7)". */
export function ratingLine(category: RatingCategory, before: number, delta: number): string {
  return `Your new ${CATEGORY_NAMES[category]} rating is ${before + delta} (${signed(delta)}).`;
}
