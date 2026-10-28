import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, vi } from "vitest";
import { useLiveClocks } from "./useLiveClocks";
import type { Game } from "../../lib/ws/schemas";

const WHITE_TO_MOVE = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1";
const BLACK_TO_MOVE =
  "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1";

const NOW = 1_700_000_000_000;

function game(overrides: Partial<Game> = {}): Game {
  return {
    id: "game-1",
    whiteId: "white-1",
    blackId: "black-1",
    fen: WHITE_TO_MOVE,
    status: "active",
    whiteTimeRemainingMs: 180_000,
    blackTimeRemainingMs: 180_000,
    initialMs: 180_000,
    incrementMs: 0,
    lastMoveAt: NOW,
    ...overrides,
  };
}

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(NOW);
});

afterEach(() => {
  vi.useRealTimers();
});

describe("useLiveClocks", () => {
  it("takes the side to move from the FEN", () => {
    const white = renderHook(() => useLiveClocks(game()));
    expect(white.result.current.activeColor).toBe("w");

    const black = renderHook(() => useLiveClocks(game({ fen: BLACK_TO_MOVE })));
    expect(black.result.current.activeColor).toBe("b");
  });

  it("reports the server's values before any time has passed", () => {
    const { result } = renderHook(() =>
      useLiveClocks(game({ whiteTimeRemainingMs: 120_000, blackTimeRemainingMs: 95_000 })),
    );

    expect(result.current.whiteMs).toBe(120_000);
    expect(result.current.blackMs).toBe(95_000);
  });

  // Nothing is pushed between moves, so the running clock has to be
  // derived from wall-clock time since lastMoveAt.
  it("counts down only the side to move", () => {
    const { result } = renderHook(() => useLiveClocks(game()));

    act(() => {
      vi.advanceTimersByTime(5_000);
    });

    expect(result.current.whiteMs).toBe(175_000);
    expect(result.current.blackMs).toBe(180_000);
  });

  it("counts down black when it is black's turn", () => {
    const { result } = renderHook(() => useLiveClocks(game({ fen: BLACK_TO_MOVE })));

    act(() => {
      vi.advanceTimersByTime(5_000);
    });

    expect(result.current.blackMs).toBe(175_000);
    expect(result.current.whiteMs).toBe(180_000);
  });

  it("clamps at zero rather than going negative", () => {
    const { result } = renderHook(() =>
      useLiveClocks(game({ whiteTimeRemainingMs: 2_000 })),
    );

    act(() => {
      vi.advanceTimersByTime(10_000);
    });

    expect(result.current.whiteMs).toBe(0);
  });

  // A finished game's clocks are history, not a countdown.
  it("stops ticking once the game is finished", () => {
    const { result } = renderHook(() =>
      useLiveClocks(game({ status: "finished", whiteTimeRemainingMs: 120_000 })),
    );

    act(() => {
      vi.advanceTimersByTime(30_000);
    });

    expect(result.current.whiteMs).toBe(120_000);
    expect(result.current.blackMs).toBe(180_000);
  });

  it("ignores a lastMoveAt in the future instead of adding time", () => {
    const { result } = renderHook(() =>
      useLiveClocks(game({ lastMoveAt: NOW + 10_000 })),
    );

    expect(result.current.whiteMs).toBe(180_000);
  });

  it("rebases when a new move arrives", () => {
    const { result, rerender } = renderHook(({ g }) => useLiveClocks(g), {
      initialProps: { g: game() },
    });

    act(() => {
      vi.advanceTimersByTime(8_000);
    });
    expect(result.current.whiteMs).toBe(172_000);

    // White moved: the server sends new clocks and it is black's turn.
    rerender({
      g: game({
        fen: BLACK_TO_MOVE,
        whiteTimeRemainingMs: 174_000,
        blackTimeRemainingMs: 180_000,
        lastMoveAt: NOW + 8_000,
      }),
    });

    expect(result.current.activeColor).toBe("b");
    expect(result.current.whiteMs).toBe(174_000);
    expect(result.current.blackMs).toBe(180_000);

    act(() => {
      vi.advanceTimersByTime(3_000);
    });

    expect(result.current.whiteMs).toBe(174_000);
    expect(result.current.blackMs).toBe(177_000);
  });
});
