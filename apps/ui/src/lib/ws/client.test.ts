import { afterEach, beforeEach, vi } from "vitest";
import {
  FakeSocket,
  fakeSocketModule,
  silenceConsoleError,
} from "../../test/fake-socket";
import { gameSocket } from "./client";
import type { ServerMessage } from "./schemas";

vi.mock("partysocket", () => fakeSocketModule());

const validMoveMade = {
  type: "game.move_made",
  payload: {
    game: {
      id: "game-1",
      whiteId: "white-1",
      blackId: "black-1",
      fen: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
      status: "active",
      whiteTimeRemainingMs: 180_000,
      blackTimeRemainingMs: 180_000,
      lastMoveAt: 1_700_000_000_000,
    },
    move: {
      gameId: "game-1",
      ply: 1,
      color: "white",
      from: "e2",
      to: "e4",
      san: "e4",
      fenAfter: "after",
      playedAt: 1_700_000_000_000,
    },
  },
};

beforeEach(() => {
  FakeSocket.reset();
});

afterEach(() => {
  // The client is a module singleton shared across tests; disconnecting
  // clears its socket so the next connect() builds a fresh one.
  gameSocket.disconnect();
  vi.restoreAllMocks();
});

describe("connect", () => {
  it("opens one socket at the /ws path", () => {
    gameSocket.connect();

    expect(FakeSocket.instances).toHaveLength(1);
    expect(FakeSocket.latest().url).toMatch(/\/ws$/);
  });

  it("reuses the existing socket when called twice", () => {
    gameSocket.connect();
    gameSocket.connect();

    expect(FakeSocket.instances).toHaveLength(1);
  });

  it("opens a new socket after a disconnect", () => {
    gameSocket.connect();
    gameSocket.disconnect();
    gameSocket.connect();

    expect(FakeSocket.instances).toHaveLength(2);
  });
});

describe("disconnect", () => {
  it("closes the underlying socket", () => {
    gameSocket.connect();
    const socket = FakeSocket.latest();

    gameSocket.disconnect();

    expect(socket.closed).toBe(true);
  });

  it("is safe with nothing connected", () => {
    expect(() => gameSocket.disconnect()).not.toThrow();
  });
});

describe("inbound messages", () => {
  it("forwards a valid envelope to subscribers", () => {
    const received: ServerMessage[] = [];
    const unsubscribe = gameSocket.subscribe((m) => received.push(m));
    gameSocket.connect();

    FakeSocket.latest().receive(validMoveMade);

    expect(received).toHaveLength(1);
    expect(received[0].type).toBe("game.move_made");
    unsubscribe();
  });

  it("delivers to every subscriber", () => {
    const first = vi.fn();
    const second = vi.fn();
    const un1 = gameSocket.subscribe(first);
    const un2 = gameSocket.subscribe(second);
    gameSocket.connect();

    FakeSocket.latest().receive(validMoveMade);

    expect(first).toHaveBeenCalledTimes(1);
    expect(second).toHaveBeenCalledTimes(1);
    un1();
    un2();
  });

  // This is the drift detector: if the Go side and these schemas disagree,
  // the message must be dropped loudly rather than reach the store as a
  // half-parsed object.
  it("rejects and logs a message that fails schema validation, without forwarding it", () => {
    const consoleError = silenceConsoleError();
    const listener = vi.fn();
    const unsubscribe = gameSocket.subscribe(listener);
    gameSocket.connect();

    FakeSocket.latest().receive({
      type: "game.move_made",
      payload: { game: { id: "game-1" } }, // missing nearly every field
    });

    expect(listener).not.toHaveBeenCalled();
    expect(consoleError).toHaveBeenCalled();
    unsubscribe();
  });

  it("rejects an unknown message type", () => {
    const consoleError = silenceConsoleError();
    const listener = vi.fn();
    const unsubscribe = gameSocket.subscribe(listener);
    gameSocket.connect();

    FakeSocket.latest().receive({ type: "game.invented", payload: {} });

    expect(listener).not.toHaveBeenCalled();
    expect(consoleError).toHaveBeenCalled();
    unsubscribe();
  });

  it("logs non-JSON frames instead of throwing", () => {
    const consoleError = silenceConsoleError();
    const listener = vi.fn();
    const unsubscribe = gameSocket.subscribe(listener);
    gameSocket.connect();

    expect(() => FakeSocket.latest().receive("this is not json")).not.toThrow();

    expect(listener).not.toHaveBeenCalled();
    expect(consoleError).toHaveBeenCalled();
    unsubscribe();
  });

  it("ignores binary frames, which this protocol never uses", () => {
    const consoleError = silenceConsoleError();
    const listener = vi.fn();
    const unsubscribe = gameSocket.subscribe(listener);
    gameSocket.connect();

    // Dropped before the JSON parse, so this is not an error worth logging -
    // unlike a string that fails to parse or validate.
    FakeSocket.latest().receiveRaw(new ArrayBuffer(8));
    FakeSocket.latest().receiveRaw({ notAString: true });

    expect(listener).not.toHaveBeenCalled();
    expect(consoleError).not.toHaveBeenCalled();
    unsubscribe();
  });
});

describe("subscribe", () => {
  it("returns an unsubscribe that stops delivery", () => {
    const listener = vi.fn();
    const unsubscribe = gameSocket.subscribe(listener);
    gameSocket.connect();

    FakeSocket.latest().receive(validMoveMade);
    expect(listener).toHaveBeenCalledTimes(1);

    unsubscribe();
    FakeSocket.latest().receive(validMoveMade);

    expect(listener).toHaveBeenCalledTimes(1);
  });
});

describe("send", () => {
  it("serialises the envelope onto the socket", () => {
    gameSocket.connect();

    gameSocket.send({
      type: "matchmaking.join",
      payload: { preset: "3+2" },
    });

    expect(FakeSocket.latest().sentEnvelopes()).toEqual([
      { type: "matchmaking.join", payload: { preset: "3+2" } },
    ]);
  });

  it("logs rather than throwing when called before connect", () => {
    const consoleError = silenceConsoleError();

    expect(() =>
      gameSocket.send({ type: "game.resign", payload: { gameId: "g1" } }),
    ).not.toThrow();

    expect(consoleError).toHaveBeenCalled();
  });
});
