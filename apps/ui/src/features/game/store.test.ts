import { beforeEach, afterEach, vi } from "vitest";
import { FakeSocket, fakeSocketModule, silenceConsoleError } from "../../test/fake-socket";
import { gameSocket } from "../../lib/ws/client";
import { useGameStore } from "./store";
import type { Game, Move } from "../../lib/ws/schemas";

vi.mock("partysocket", () => fakeSocketModule());

function game(overrides: Partial<Game> = {}): Game {
  return {
    id: "game-1",
    whiteId: "white-1",
    blackId: "black-1",
    fen: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
    status: "active",
    whiteTimeRemainingMs: 180_000,
    blackTimeRemainingMs: 180_000,
    initialMs: 180_000,
    incrementMs: 0,
    lastMoveAt: 1_700_000_000_000,
    ...overrides,
  };
}

function move(ply: number, san: string): Move {
  return {
    gameId: "game-1",
    ply,
    color: ply % 2 === 1 ? "white" : "black",
    from: "e2",
    to: "e4",
    san,
    fenAfter: "after",
    playedAt: 1_700_000_000_000 + ply,
  };
}

const initialState = {
  currentGame: null,
  moves: [],
  lastMove: null,
  matchmakingStatus: "idle" as const,
  matchedGameId: null,
  opponent: null,
  queuedPreset: null,
};

beforeEach(() => {
  FakeSocket.reset();
  useGameStore.setState(initialState);
  gameSocket.connect();
});

afterEach(() => {
  gameSocket.disconnect();
  vi.restoreAllMocks();
});

/** What the socket was asked to send, newest last. */
function sent() {
  return FakeSocket.latest().sentEnvelopes();
}

describe("inbound game.move_made", () => {
  it("stores the game, the move, and appends to the history", () => {
    FakeSocket.latest().receive({
      type: "game.move_made",
      payload: { game: game({ fen: "after-e4" }), move: move(1, "e4") },
    });

    const state = useGameStore.getState();
    expect(state.currentGame?.fen).toBe("after-e4");
    expect(state.lastMove?.san).toBe("e4");
    expect(state.moves.map((m) => m.san)).toEqual(["e4"]);
  });

  it("accumulates moves across several frames", () => {
    const socket = FakeSocket.latest();

    socket.receive({ type: "game.move_made", payload: { game: game(), move: move(1, "e4") } });
    socket.receive({ type: "game.move_made", payload: { game: game(), move: move(2, "e5") } });
    socket.receive({ type: "game.move_made", payload: { game: game(), move: move(3, "Nf3") } });

    expect(useGameStore.getState().moves.map((m) => m.san)).toEqual(["e4", "e5", "Nf3"]);
  });

  // The server resends full state, so a reconnect can redeliver a frame.
  it("replaces rather than duplicates a redelivered ply", () => {
    const socket = FakeSocket.latest();

    socket.receive({ type: "game.move_made", payload: { game: game(), move: move(1, "e4") } });
    socket.receive({ type: "game.move_made", payload: { game: game(), move: move(1, "e4") } });

    expect(useGameStore.getState().moves).toHaveLength(1);
  });
});

describe("inbound game.game_over", () => {
  it("stores the finished game", () => {
    FakeSocket.latest().receive({
      type: "game.game_over",
      payload: { game: game({ status: "finished", result: "white_wins", resultReason: "checkmate" }) },
    });

    const state = useGameStore.getState();
    expect(state.currentGame?.status).toBe("finished");
    expect(state.currentGame?.result).toBe("white_wins");
  });
});

describe("inbound matchmaking.matched", () => {
  it("records the opponent and the game to navigate to", () => {
    FakeSocket.latest().receive({
      type: "matchmaking.matched",
      payload: { gameId: "game-9", opponent: { username: "bob", rating: 1337 } },
    });

    const state = useGameStore.getState();
    expect(state.matchmakingStatus).toBe("matched");
    expect(state.matchedGameId).toBe("game-9");
    expect(state.opponent).toEqual({ username: "bob", rating: 1337 });
  });
});

describe("inbound error", () => {
  it("logs without disturbing game state", () => {
    const consoleError = silenceConsoleError();
    useGameStore.setState({ currentGame: game() });

    FakeSocket.latest().receive({
      type: "error",
      payload: { code: "illegal_move", message: "illegal move" },
    });

    expect(consoleError).toHaveBeenCalled();
    expect(useGameStore.getState().currentGame).not.toBeNull();
  });
});

describe("joinQueue / leaveQueue", () => {
  it("sends the join envelope and marks the status queued", () => {
    useGameStore.getState().joinQueue("3+2");

    expect(sent()).toEqual([{ type: "matchmaking.join", payload: { preset: "3+2" } }]);
    expect(useGameStore.getState().matchmakingStatus).toBe("queued");
    expect(useGameStore.getState().queuedPreset).toBe("3+2");
  });

  // Regression: joinQueue never recorded the preset, so this sent nothing.
  it("join then leave sends leave for the same preset", () => {
    useGameStore.getState().joinQueue("10+5");

    useGameStore.getState().leaveQueue();

    expect(sent().at(-1)).toEqual({ type: "matchmaking.leave", payload: { preset: "10+5" } });
    expect(useGameStore.getState().queuedPreset).toBeNull();
  });

  it("a refused join stops searching", () => {
    const consoleError = silenceConsoleError();
    useGameStore.getState().joinQueue("3+2");

    FakeSocket.latest().receive({
      type: "error",
      payload: { code: "already_in_game", message: "already in a game", inReplyTo: "matchmaking.join" },
    });

    expect(useGameStore.getState().matchmakingStatus).toBe("idle");
    expect(useGameStore.getState().queuedPreset).toBeNull();
    consoleError.mockRestore();
  });

  it("sends leave for the queued preset", () => {
    useGameStore.setState({ queuedPreset: "3+2", matchmakingStatus: "queued" });

    useGameStore.getState().leaveQueue();

    expect(sent()).toEqual([{ type: "matchmaking.leave", payload: { preset: "3+2" } }]);
    expect(useGameStore.getState().matchmakingStatus).toBe("idle");
  });

  // Nothing to leave: sending a preset-less leave would be meaningless.
  it("does nothing when no preset is queued", () => {
    useGameStore.getState().leaveQueue();

    expect(sent()).toEqual([]);
  });
});

describe("submitMove", () => {
  it("sends the current game's id with the move", () => {
    useGameStore.setState({ currentGame: game() });

    useGameStore.getState().submitMove("e2", "e4");

    expect(sent()).toEqual([
      { type: "game.submit_move", payload: { gameId: "game-1", from: "e2", to: "e4" } },
    ]);
  });

  it("includes a promotion when given", () => {
    useGameStore.setState({ currentGame: game() });

    useGameStore.getState().submitMove("a7", "a8", "q");

    expect(sent()[0].payload).toMatchObject({ from: "a7", to: "a8", promotion: "q" });
  });

  it("does nothing with no game loaded", () => {
    useGameStore.getState().submitMove("e2", "e4");

    expect(sent()).toEqual([]);
  });
});

describe("resign", () => {
  it("sends the current game's id", () => {
    useGameStore.setState({ currentGame: game() });

    useGameStore.getState().resign();

    expect(sent()).toEqual([{ type: "game.resign", payload: { gameId: "game-1" } }]);
  });

  it("does nothing with no game loaded", () => {
    useGameStore.getState().resign();

    expect(sent()).toEqual([]);
  });
});

describe("subscribeToGame", () => {
  it("sends the subscribe envelope for the given id", () => {
    useGameStore.getState().subscribeToGame("game-7");

    expect(sent()).toEqual([{ type: "game.subscribe_game", payload: { gameId: "game-7" } }]);
  });
});

describe("setInitialGame / setInitialMoves", () => {
  it("seeds the game from a REST fetch", () => {
    useGameStore.getState().setInitialGame(game({ fen: "seeded" }));

    expect(useGameStore.getState().currentGame?.fen).toBe("seeded");
  });

  // The fetch can land after a live frame has already arrived; the live
  // move must survive the seed.
  it("keeps a live move that arrived while the fetch was in flight", () => {
    FakeSocket.latest().receive({
      type: "game.move_made",
      payload: { game: game(), move: move(2, "e5") },
    });

    useGameStore.getState().setInitialMoves([move(1, "e4")]);

    expect(useGameStore.getState().moves.map((m) => m.ply)).toEqual([1, 2]);
  });
});

describe("clearMatch", () => {
  it("resets matchmaking state and the move history", () => {
    useGameStore.setState({
      matchmakingStatus: "matched",
      matchedGameId: "game-1",
      opponent: { username: "bob", rating: 1200 },
      moves: [move(1, "e4")],
    });

    useGameStore.getState().clearMatch();

    const state = useGameStore.getState();
    expect(state.matchmakingStatus).toBe("idle");
    expect(state.matchedGameId).toBeNull();
    expect(state.opponent).toBeNull();
    expect(state.moves).toEqual([]);
  });
});
