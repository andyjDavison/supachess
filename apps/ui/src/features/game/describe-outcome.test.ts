import { describeOutcome } from "./describe-outcome";
import type { Game } from "../../lib/ws/schemas";

function finishedGame(overrides: Partial<Game>): Game {
  return {
    id: "game-1",
    whiteId: "white-1",
    blackId: "black-1",
    fen: "8/8/8/8/8/8/8/8 w - - 0 1",
    status: "finished",
    whiteTimeRemainingMs: 1000,
    blackTimeRemainingMs: 1000,
    lastMoveAt: 0,
    ...overrides,
  };
}

describe("describeOutcome", () => {
  describe("decisive games", () => {
    it("tells the winner they won, from either colour", () => {
      const whiteWon = finishedGame({ result: "white_wins", resultReason: "checkmate" });

      expect(describeOutcome(whiteWon, true).headline).toBe("You won!");
      expect(describeOutcome(whiteWon, false).headline).toBe("You lost");

      const blackWon = finishedGame({ result: "black_wins", resultReason: "checkmate" });

      expect(describeOutcome(blackWon, false).headline).toBe("You won!");
      expect(describeOutcome(blackWon, true).headline).toBe("You lost");
    });

    it("describes how the game was decided", () => {
      const cases: Array<[string, string, string]> = [
        ["checkmate", "Won by checkmate", "Lost by checkmate"],
        ["resignation", "Won by resignation", "Lost by resignation"],
        ["timeout", "Won on time", "Lost on time"],
      ];

      for (const [reason, winnerDetail, loserDetail] of cases) {
        const game = finishedGame({ result: "white_wins", resultReason: reason });

        expect(describeOutcome(game, true).detail).toBe(winnerDetail);
        expect(describeOutcome(game, false).detail).toBe(loserDetail);
      }
    });

    it("omits the detail for an unrecognised reason rather than inventing one", () => {
      const game = finishedGame({ result: "white_wins", resultReason: "abandoned" });

      expect(describeOutcome(game, true).headline).toBe("You won!");
      expect(describeOutcome(game, true).detail).toBe("");
    });
  });

  describe("draws", () => {
    it("reads as a draw for both players", () => {
      const game = finishedGame({ result: "draw", resultReason: "stalemate" });

      expect(describeOutcome(game, true).headline).toBe("Draw");
      expect(describeOutcome(game, false).headline).toBe("Draw");
    });

    it("names each draw reason", () => {
      const cases: Record<string, string> = {
        stalemate: "Stalemate",
        threefold_repetition: "Draw by repetition",
        fifty_move_rule: "Draw by the fifty-move rule",
        insufficient_material: "Draw by insufficient material",
        draw_agreement: "Draw by agreement",
      };

      for (const [reason, expected] of Object.entries(cases)) {
        const game = finishedGame({ result: "draw", resultReason: reason });
        expect(describeOutcome(game, true).detail).toBe(expected);
      }
    });

    it("falls back to generic wording for an unknown draw reason", () => {
      const game = finishedGame({ result: "draw", resultReason: "mutual_boredom" });

      expect(describeOutcome(game, true).detail).toBe("The game ended in a draw");
    });

    it("falls back when the reason is missing entirely", () => {
      const game = finishedGame({ result: "draw" });

      expect(describeOutcome(game, true).detail).toBe("The game ended in a draw");
    });
  });
});
