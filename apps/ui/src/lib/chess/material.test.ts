import { describe, expect, it } from "vitest";
import { material } from "./material";

const START = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1";

describe("material", () => {
  it("has nothing captured at the start", () => {
    expect(material(START)).toEqual({ capturedByWhite: [], capturedByBlack: [], balance: 0 });
  });

  it("lists a capture under the side that made it", () => {
    // 1.e4 d5 2.exd5: White has taken a pawn.
    const m = material("rnbqkbnr/ppp1pppp/8/3P4/8/8/PPPP1PPP/RNBQKBNR b KQkq - 0 2");
    expect(m.capturedByWhite).toEqual(["p"]);
    expect(m.capturedByBlack).toEqual([]);
    expect(m.balance).toBe(1);
  });

  it("sorts captures cheapest first and nets out an even trade", () => {
    // White has taken a knight and a pawn; Black has taken a bishop and a pawn.
    const m = material("r1bqkbnr/ppp1pppp/8/8/8/8/PPPP1PPP/RN1QKBNR w KQkq - 0 5");
    expect(m.capturedByWhite).toEqual(["p", "n"]);
    expect(m.capturedByBlack).toEqual(["p", "b"]);
    expect(m.balance).toBe(0);
  });

  it("counts a promotion as the pawn it was, not a captured pawn", () => {
    // White promoted a pawn to a second queen; nothing was captured from White.
    const m = material("Q3k3/8/8/8/8/8/PPPPPPP1/RNBQKBNR w - - 0 40");
    expect(m.capturedByBlack).not.toContain("p");
    // ...and Black has lost everything but the king.
    expect(m.capturedByWhite).toEqual(["p", "p", "p", "p", "p", "p", "p", "p", "n", "n", "b", "b", "r", "r", "q"]);
    // 7 pawns, two knights, two bishops, two rooks, two queens.
    expect(m.balance).toBe(7 + 6 + 6 + 10 + 18);
  });
});
