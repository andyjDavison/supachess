// What each side has captured, read off a FEN. There's no capture history
// to replay: comparing the board with the starting set is enough, and it
// works the same for a game loaded mid-way.

export type PieceType = "p" | "n" | "b" | "r" | "q";

const ORDER: PieceType[] = ["p", "n", "b", "r", "q"];
const VALUE: Record<PieceType, number> = { p: 1, n: 3, b: 3, r: 5, q: 9 };
const START: Record<PieceType, number> = { p: 8, n: 2, b: 2, r: 2, q: 1 };

export interface Material {
  /** Black pieces White has taken, cheapest first. */
  capturedByWhite: PieceType[];
  /** White pieces Black has taken, cheapest first. */
  capturedByBlack: PieceType[];
  /** White's material minus Black's: positive means White is ahead. */
  balance: number;
}

function count(placement: string, isWhite: boolean): Record<PieceType, number> {
  const counts: Record<PieceType, number> = { p: 0, n: 0, b: 0, r: 0, q: 0 };
  for (const ch of placement) {
    const lower = ch.toLowerCase() as PieceType;
    if (lower in counts && (ch !== lower) === isWhite) counts[lower]++;
  }
  return counts;
}

// A promoted piece is an extra over the starting set; it came from a pawn
// that's gone but wasn't captured, so it cancels one missing pawn.
function missing(counts: Record<PieceType, number>): PieceType[] {
  const extras = ORDER.slice(1).reduce((n, t) => n + Math.max(0, counts[t] - START[t]), 0);
  const out: PieceType[] = [];
  for (const t of ORDER) {
    let n = Math.max(0, START[t] - counts[t]);
    if (t === "p") n = Math.max(0, n - extras);
    for (let i = 0; i < n; i++) out.push(t);
  }
  return out;
}

function value(counts: Record<PieceType, number>): number {
  return ORDER.reduce((sum, t) => sum + counts[t] * VALUE[t], 0);
}

export function material(fen: string): Material {
  const placement = fen.split(" ")[0] ?? "";
  const white = count(placement, true);
  const black = count(placement, false);
  return {
    capturedByWhite: missing(black),
    capturedByBlack: missing(white),
    balance: value(white) - value(black),
  };
}
