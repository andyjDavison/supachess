import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FakeSocket, fakeSocketModule } from "../../test/fake-socket";
import { gameSocket } from "../../lib/ws/client";
import type { Game, GameMessage } from "../../lib/ws/schemas";
import { renderWithProviders } from "../../test/render";
import { GameChat } from "./game-chat";
import { describeGameEnd, formatStakes, formatTimeControl, ratingLine } from "./game-chat-text";

vi.mock("partysocket", () => fakeSocketModule());

const me = { id: "white-1", email: "w@example.com", username: "white" };

function game(overrides: Partial<Game> = {}): Game {
  return {
    id: "game-1",
    whiteId: "white-1",
    blackId: "black-1",
    fen: "start",
    status: "active",
    whiteTimeRemainingMs: 180_000,
    blackTimeRemainingMs: 180_000,
    initialMs: 180_000,
    incrementMs: 0,
    lastMoveAt: 1_700_000_000_000,
    ...overrides,
  };
}

function gm(id: number, senderId: string, body: string): GameMessage {
  return { id, gameId: "game-1", senderId, body, sentAt: 1_700_000_000_000 + id };
}

function stubApi(log: GameMessage[], user = me) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string) => {
      if (url.includes("/api/me")) return { ok: true, status: 200, json: async () => user };
      if (url.includes("/api/games/game-1/chat")) return { ok: true, status: 200, json: async () => ({ messages: log }) };
      if (url.includes("/api/user/white-1")) return { ok: true, status: 200, json: async () => ({ id: "white-1", username: "i_luv_milk", rating: 103 }) };
      if (url.includes("/api/user/black-1")) return { ok: true, status: 200, json: async () => ({ id: "black-1", username: "El_Aspany1", rating: 116 }) };
      throw new Error(`unexpected ${url}`);
    }),
  );
}

beforeEach(() => {
  FakeSocket.reset();
  gameSocket.connect();
});

afterEach(() => {
  gameSocket.disconnect();
  vi.unstubAllGlobals();
});

const socket = () => FakeSocket.latest();

describe("GameChat", () => {
  it("opens with a New Game line: players, ratings, time control", async () => {
    stubApi([]);

    renderWithProviders(<GameChat game={game({ initialMs: 60_000, incrementMs: 0 })} />);

    const log = screen.getByRole("log", { name: "Game chat" });
    await waitFor(() => expect(log).toHaveTextContent(/New Game\s*i_luv_milk \(103\) vs\. El_Aspany1 \(116\) \(1 min\)/));
    expect(log).not.toHaveTextContent("Game Over");
  });

  it("uses the game's ratings, and shows a player their stakes", async () => {
    stubApi([]);
    const ratings = {
      category: "bullet" as const,
      white: { rating: 1500, win: 18, draw: 8, loss: -2 },
      black: { rating: 1900, win: 2, draw: -8, loss: -18 },
    };

    renderWithProviders(<GameChat game={game({ initialMs: 60_000, ratings })} />);

    const log = screen.getByRole("log", { name: "Game chat" });
    await waitFor(() =>
      expect(log).toHaveTextContent(/i_luv_milk \(1500\) vs\. El_Aspany1 \(1900\) \(1 min\)\s*win \+18 \/ draw \+8 \/ lose -2/),
    );
  });

  it("tells a player their new rating at the end", async () => {
    stubApi([]);
    const ratings = {
      category: "bullet" as const,
      white: { rating: 1500, win: 18, draw: 8, loss: -2, delta: -2 },
      black: { rating: 1900, win: 2, draw: -8, loss: -18, delta: 2 },
    };

    renderWithProviders(
      <GameChat game={game({ status: "finished", result: "black_wins", resultReason: "timeout", initialMs: 60_000, ratings })} />,
    );

    const log = screen.getByRole("log", { name: "Game chat" });
    await waitFor(() => expect(log).toHaveTextContent(/El_Aspany1 won on time \(1 min\)\s*Your new Bullet rating is 1498 \(-2\)\.$/));
  });

  it("shows a spectator the ratings but no personal lines", async () => {
    const spectator = { id: "someone", email: "s@example.com", username: "watcher" };
    stubApi([], spectator);
    const ratings = {
      category: "blitz" as const,
      white: { rating: 1500, win: 18, draw: 8, loss: -2, delta: 18 },
      black: { rating: 1900, win: 2, draw: -8, loss: -18, delta: -18 },
    };

    renderWithProviders(<GameChat game={game({ status: "finished", result: "white_wins", resultReason: "checkmate", ratings })} />);

    const log = screen.getByRole("log", { name: "Game chat" });
    await waitFor(() => expect(log).toHaveTextContent("i_luv_milk (1500)"));
    expect(log).not.toHaveTextContent("win +");
    expect(log).not.toHaveTextContent("Your new");
  });

  it("loads the log, labelling each message with its sender", async () => {
    stubApi([gm(1, "black-1", "good luck"), gm(2, "white-1", "you too")]);

    renderWithProviders(<GameChat game={game()} />);

    expect(await screen.findByText("good luck")).toBeInTheDocument();
    const log = screen.getByRole("log", { name: "Game chat" });
    await waitFor(() => expect(log).toHaveTextContent("El_Aspany1: good luck"));
    expect(log).toHaveTextContent("i_luv_milk: you too");
  });

  it("closes with a Game Over line once the game is finished", async () => {
    stubApi([gm(1, "black-1", "gg")]);

    renderWithProviders(
      <GameChat
        game={game({ status: "finished", result: "black_wins", resultReason: "checkmate", initialMs: 180_000, incrementMs: 2_000 })}
      />,
    );

    const log = screen.getByRole("log", { name: "Game chat" });
    await waitFor(() => expect(log).toHaveTextContent(/gg\s*Game Over\s*El_Aspany1 won by checkmate \(3 \| 2\)$/));
  });

  it("appends live messages for this game only", async () => {
    stubApi([]);
    renderWithProviders(<GameChat game={game()} />);
    await screen.findByText(/New Game/);

    act(() => {
      socket().receive({ type: "chat.game_message", payload: { gameId: "game-1", message: gm(5, "black-1", "nice move") } });
      socket().receive({
        type: "chat.game_message",
        payload: { gameId: "game-2", message: { ...gm(6, "x", "elsewhere"), gameId: "game-2" } },
      });
    });

    expect(screen.getByText("nice move")).toBeInTheDocument();
    expect(screen.queryByText("elsewhere")).not.toBeInTheDocument();
  });

  it("sends chat.send_game", async () => {
    stubApi([]);
    renderWithProviders(<GameChat game={game()} />);
    const box = await screen.findByLabelText("Message");
    await waitFor(() => expect(box).toBeEnabled());

    await userEvent.type(box, "gg{Enter}");

    expect(socket().sentEnvelopes()).toContainEqual({ type: "chat.send_game", payload: { gameId: "game-1", body: "gg" } });
  });

  it("is read-only once the game is over", async () => {
    stubApi([gm(1, "black-1", "gg wp")]);

    renderWithProviders(<GameChat game={game({ status: "finished" })} />);

    expect(await screen.findByText("gg wp")).toBeInTheDocument();
    expect(await screen.findByText("Chat closed.")).toBeInTheDocument();
    expect(screen.queryByLabelText("Message")).not.toBeInTheDocument();
  });

  it("shows a refused send", async () => {
    stubApi([]);
    renderWithProviders(<GameChat game={game()} />);
    await screen.findByText(/New Game/);

    act(() =>
      socket().receive({
        type: "error",
        payload: { code: "rate_limited", message: "slow down", inReplyTo: "chat.send_game" },
      }),
    );

    expect(screen.getByRole("alert")).toHaveTextContent("too fast");
  });
});

describe("game chat text", () => {
  it.each([
    [60_000, 0, "1 min"],
    [600_000, 0, "10 min"],
    [180_000, 2_000, "3 | 2"],
    [900_000, 10_000, "15 | 10"],
  ])("formatTimeControl(%d, %d) = %s", (initial, increment, want) => {
    expect(formatTimeControl(initial, increment)).toBe(want);
  });

  it.each([
    [{ result: "white_wins", resultReason: "resignation" }, "alice won by resignation"],
    [{ result: "black_wins", resultReason: "timeout" }, "bob won on time"],
    [{ result: "draw", resultReason: "stalemate" }, "Game drawn by stalemate"],
    [{ result: "draw", resultReason: "threefold_repetition" }, "Game drawn by repetition"],
    [{ result: "draw" }, "Game drawn"],
  ])("describeGameEnd(%o) = %s", (outcome, want) => {
    expect(describeGameEnd(game({ status: "finished", ...outcome }), "alice", "bob")).toBe(want);
  });

  it("signs every stake, including zero", () => {
    expect(formatStakes({ rating: 1200, win: 20, draw: 0, loss: -20 })).toBe("win +20 / draw +0 / lose -20");
  });

  it.each([
    ["rapid", 1200, 7, "Your new Rapid rating is 1207 (+7)."],
    ["blitz", 1200, -12, "Your new Blitz rating is 1188 (-12)."],
    ["bullet", 1200, 0, "Your new Bullet rating is 1200 (+0)."],
  ] as const)("ratingLine(%s, %d, %d)", (category, before, delta, want) => {
    expect(ratingLine(category, before, delta)).toBe(want);
  });
});
