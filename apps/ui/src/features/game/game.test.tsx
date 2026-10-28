import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router";
import GamePage from "./game";
import { useGameStore } from "./store";
import { AuthProvider } from "../../stores/AuthContext";
import { AuthModalProvider } from "../../stores/AuthModalContext";
import type { Game } from "../../lib/ws/schemas";

vi.mock("../../lib/ws/client", () => ({
  gameSocket: { connect: vi.fn(), disconnect: vi.fn(), subscribe: vi.fn(() => () => {}), send: vi.fn() },
}));

// The board renders react-chessboard, which is not what this file is about.
vi.mock("./game-board", () => ({
  GameBoard: ({ game }: { game: Game }) => <div data-testid="board">{game.fen}</div>,
}));

const whiteUser = { id: "white-1", email: "w@example.com", username: "white" };

const activeGame = {
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
};

const moveRow = {
  gameId: "game-1",
  ply: 1,
  color: "white",
  from: "e2",
  to: "e4",
  san: "e4",
  fenAfter: "after",
  playedAt: 1_700_000_000_000,
};

/** Routes /api/me, the game fetch and the moves fetch by URL. */
function stubApi({
  game = activeGame as unknown,
  moves = [moveRow] as unknown,
  gameOk = true,
  movesOk = true,
}: { game?: unknown; moves?: unknown; gameOk?: boolean; movesOk?: boolean } = {}) {
  const fetchMock = vi.fn(async (url: string) => {
    if (url.includes("/api/me")) {
      return { ok: true, status: 200, json: async () => whiteUser };
    }
    if (url.includes("/moves")) {
      return { ok: movesOk, status: movesOk ? 200 : 500, json: async () => moves };
    }
    return { ok: gameOk, status: gameOk ? 200 : 404, json: async () => game };
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function renderGamePage() {
  return render(
    <MemoryRouter initialEntries={["/game/game-1"]}>
      <AuthProvider>
        <AuthModalProvider>
          <Routes>
            <Route path="/game/:gameId" element={<GamePage />} />
          </Routes>
        </AuthModalProvider>
      </AuthProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  useGameStore.setState({ currentGame: null, moves: [], lastMove: null });
});

describe("GamePage", () => {
  it("shows a loading state until the game arrives", () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise(() => {})));

    renderGamePage();

    expect(screen.getByText(/Loading game/)).toBeInTheDocument();
  });

  it("fetches the game and its move history on mount", async () => {
    const fetchMock = stubApi();

    renderGamePage();

    await waitFor(() => expect(screen.getByTestId("board")).toBeInTheDocument());

    const urls = fetchMock.mock.calls.map(([url]) => url as string);
    expect(urls.some((u) => u.endsWith("/api/games/game-1"))).toBe(true);
    expect(urls.some((u) => u.endsWith("/api/games/game-1/moves"))).toBe(true);
  });

  it("seeds the store with the fetched game and moves", async () => {
    stubApi();

    renderGamePage();

    await waitFor(() => expect(useGameStore.getState().currentGame?.id).toBe("game-1"));
    await waitFor(() => expect(useGameStore.getState().moves.map((m) => m.san)).toEqual(["e4"]));
  });

  it("shows an error when the game cannot be loaded", async () => {
    stubApi({ gameOk: false });

    renderGamePage();

    await waitFor(() => expect(screen.getByText(/Couldn't load this game/)).toBeInTheDocument());
  });

  // Move history is supplementary: losing it must not blank the page.
  it("still renders the board when the move fetch fails", async () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    stubApi({ movesOk: false });

    renderGamePage();

    await waitFor(() => expect(screen.getByTestId("board")).toBeInTheDocument());
    expect(screen.queryByText(/Couldn't load this game/)).not.toBeInTheDocument();
    consoleError.mockRestore();
  });

  it("opens on the Analysis tab with the move list visible", async () => {
    stubApi();

    renderGamePage();

    // Defaulting to a blank panel would hide the move list on load.
    await waitFor(() => expect(screen.getByText("e4")).toBeInTheDocument());
  });

  it("switches panels when another tab is clicked", async () => {
    stubApi();
    renderGamePage();
    await waitFor(() => expect(screen.getByText("e4")).toBeInTheDocument());

    await userEvent.click(screen.getByRole("button", { name: "Players" }));

    expect(screen.queryByText("e4")).not.toBeInTheDocument();
  });

  it("exposes all four tabs as buttons", async () => {
    stubApi();
    renderGamePage();
    await waitFor(() => expect(screen.getByTestId("board")).toBeInTheDocument());

    for (const tab of ["Analysis", "New Game", "Games", "Players"]) {
      expect(screen.getByRole("button", { name: tab })).toBeInTheDocument();
    }
  });

  it("shows the game-over modal for a finished game", async () => {
    stubApi({
      game: { ...activeGame, status: "finished", result: "white_wins", resultReason: "checkmate" },
    });

    renderGamePage();

    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());
    expect(screen.getByText("You won!")).toBeInTheDocument();
  });
});
