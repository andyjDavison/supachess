import type { Move } from "../../lib/ws/schemas";

export interface MoveRow {
  number: number;
  white?: Move;
  black?: Move;
}

// pairMoves groups a ply-ordered list into the numbered rows a chess move
// list displays - move 1 being white's ply 1 plus black's ply 2.
//
// The row number comes from the ply, but the column comes from the move's
// own `color` field rather than ply parity: color is what the server
// actually asserts, so a list that somehow starts on black renders in the
// right column instead of silently shifting every pair.
//
// Lives apart from move-list.tsx so that file exports only its component,
// which is what react-refresh needs to hot-reload it.
export function pairMoves(moves: Move[]): MoveRow[] {
  const rows = new Map<number, MoveRow>();

  for (const move of moves) {
    const number = Math.ceil(move.ply / 2);
    const row = rows.get(number) ?? { number };
    if (move.color === "white") {
      row.white = move;
    } else {
      row.black = move;
    }
    rows.set(number, row);
  }

  return [...rows.values()].sort((a, b) => a.number - b.number);
}
