import { act, render, screen } from "@testing-library/react";
import { MoveList } from "./move-list";
import { pairMoves } from "./move-rows";
import { mergeMoves, useGameStore } from "./store";
import type { Move } from "../../lib/ws/schemas";

function move(ply: number, san: string): Move {
  return {
    gameId: "g1",
    ply,
    color: ply % 2 === 1 ? "white" : "black",
    from: "e2",
    to: "e4",
    san,
    fenAfter: "",
    playedAt: ply,
  };
}

describe("pairMoves", () => {
  it("pairs white and black plies into one numbered row", () => {
    const rows = pairMoves([move(1, "e4"), move(2, "e5")]);

    expect(rows).toHaveLength(1);
    expect(rows[0].number).toBe(1);
    expect(rows[0].white?.san).toBe("e4");
    expect(rows[0].black?.san).toBe("e5");
  });

  it("leaves black undefined on a trailing white move", () => {
    const rows = pairMoves([move(1, "e4"), move(2, "e5"), move(3, "Nf3")]);

    expect(rows).toHaveLength(2);
    expect(rows[1].white?.san).toBe("Nf3");
    expect(rows[1].black).toBeUndefined();
  });

  it("numbers rows correctly past move 9", () => {
    const moves = Array.from({ length: 24 }, (_, i) => move(i + 1, `m${i + 1}`));
    const rows = pairMoves(moves);

    expect(rows).toHaveLength(12);
    expect(rows[11].number).toBe(12);
    expect(rows[11].white?.san).toBe("m23");
    expect(rows[11].black?.san).toBe("m24");
  });

  it("places by the move's own color, not ply parity", () => {
    // A history that starts on black - impossible from the standard
    // position, but the column must follow `color` regardless.
    const blackFirst: Move = { ...move(1, "e5"), color: "black" };
    const rows = pairMoves([blackFirst]);

    expect(rows[0].black?.san).toBe("e5");
    expect(rows[0].white).toBeUndefined();
  });

  it("returns no rows for an empty history", () => {
    expect(pairMoves([])).toEqual([]);
  });
});

describe("mergeMoves", () => {
  it("appends a new move in ply order", () => {
    const merged = mergeMoves([move(1, "e4")], [move(2, "e5")]);

    expect(merged.map((m) => m.san)).toEqual(["e4", "e5"]);
  });

  it("replaces rather than duplicates on a repeated ply", () => {
    const merged = mergeMoves([move(1, "e4")], [move(1, "d4")]);

    expect(merged).toHaveLength(1);
    expect(merged[0].san).toBe("d4");
  });

  it("sorts out-of-order input by ply", () => {
    const merged = mergeMoves([move(3, "Nf3"), move(1, "e4")], [move(2, "e5")]);

    expect(merged.map((m) => m.ply)).toEqual([1, 2, 3]);
  });

  it("keeps live moves when seeding a fetched history", () => {
    // setInitialMoves passes the fetched list first, so a move that
    // arrived over the socket mid-flight survives.
    const fetched = [move(1, "e4")];
    const live = [move(2, "e5")];

    expect(mergeMoves(fetched, live).map((m) => m.ply)).toEqual([1, 2]);
  });

  it("handles both sides being empty", () => {
    expect(mergeMoves([], [])).toEqual([]);
  });
});

describe("MoveList", () => {
  beforeEach(() => {
    useGameStore.setState({ moves: [] });
  });

  it("shows an empty state before any move is played", () => {
    render(<MoveList />);

    expect(screen.getByText("No moves yet")).toBeInTheDocument();
  });

  it("renders a row per move pair, numbered and in order", () => {
    useGameStore.setState({
      moves: [move(1, "e4"), move(2, "e5"), move(3, "Nf3")],
    });

    render(<MoveList />);

    expect(screen.queryByText("No moves yet")).not.toBeInTheDocument();

    // Two pair rows. The scroll anchor is aria-hidden, so it is
    // correctly absent from the accessibility tree.
    const rows = screen.getAllByRole("listitem");
    expect(rows).toHaveLength(2);
    expect(rows[0].textContent).toBe("1.e4e5");
    expect(rows[1].textContent).toBe("2.Nf3");
  });

  it("picks up a move appended to the store after mount", () => {
    useGameStore.setState({ moves: [move(1, "e4")] });

    render(<MoveList />);
    expect(screen.getByText("e4")).toBeInTheDocument();

    // What the game.move_made handler effectively does.
    act(() => {
      useGameStore.setState((s) => ({ moves: mergeMoves(s.moves, [move(2, "c5")]) }));
    });

    expect(screen.getByText("c5")).toBeInTheDocument();
  });
});
