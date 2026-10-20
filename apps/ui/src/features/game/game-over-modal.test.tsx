import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, vi } from "vitest";
import { GameOverModal } from "./game-over-modal";
import { useGameStore } from "./store";
import { renderWithProviders } from "../../test/render";
import type { Game } from "../../lib/ws/schemas";

vi.mock("../../lib/ws/client", () => ({
  gameSocket: { connect: vi.fn(), disconnect: vi.fn(), subscribe: vi.fn(() => () => {}), send: vi.fn() },
}));

const navigate = vi.fn();
vi.mock("react-router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-router")>()),
  useNavigate: () => navigate,
}));

const whiteUser = { id: "white-1", email: "w@example.com", username: "white" };

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

/** Signed in as white. */
function stubSignedIn() {
  vi.stubGlobal("fetch", vi.fn(async () => ({ ok: true, status: 200, json: async () => whiteUser })));
}

beforeEach(() => {
  navigate.mockClear();
  stubSignedIn();
});

describe("GameOverModal", () => {
  it("tells white they won when white wins", async () => {
    renderWithProviders(
      <GameOverModal game={finishedGame({ result: "white_wins", resultReason: "checkmate" })} />,
    );

    await waitFor(() => expect(screen.getByText("You won!")).toBeInTheDocument());
    expect(screen.getByText("Won by checkmate")).toBeInTheDocument();
  });

  it("tells white they lost when black wins", async () => {
    renderWithProviders(
      <GameOverModal game={finishedGame({ result: "black_wins", resultReason: "resignation" })} />,
    );

    await waitFor(() => expect(screen.getByText("You lost")).toBeInTheDocument());
    expect(screen.getByText("Lost by resignation")).toBeInTheDocument();
  });

  it("shows a draw with its reason", async () => {
    renderWithProviders(
      <GameOverModal game={finishedGame({ result: "draw", resultReason: "stalemate" })} />,
    );

    await waitFor(() => expect(screen.getByText("Draw")).toBeInTheDocument());
    expect(screen.getByText("Stalemate")).toBeInTheDocument();
  });

  it("is announced as a modal dialog", async () => {
    renderWithProviders(
      <GameOverModal game={finishedGame({ result: "white_wins", resultReason: "checkmate" })} />,
    );

    const dialog = await screen.findByRole("dialog");
    expect(dialog).toHaveAttribute("aria-modal", "true");
    // The headline names the dialog for assistive tech.
    expect(dialog).toHaveAttribute("aria-labelledby", "game-over-title");
  });

  it("clears match state and returns to the lobby", async () => {
    useGameStore.setState({
      matchmakingStatus: "matched",
      matchedGameId: "game-1",
      opponent: { username: "black", rating: 1200 },
    });
    renderWithProviders(
      <GameOverModal game={finishedGame({ result: "white_wins", resultReason: "checkmate" })} />,
    );

    await userEvent.click(await screen.findByRole("button", { name: "Back to lobby" }));

    expect(navigate).toHaveBeenCalledWith("/play/online");
    // Leaving with stale matchmaking state would re-trigger a navigation.
    const state = useGameStore.getState();
    expect(state.matchmakingStatus).toBe("idle");
    expect(state.matchedGameId).toBeNull();
    expect(state.opponent).toBeNull();
  });

  // Known placeholder: the button exists but does nothing yet.
  it("renders an Analyze button that does not navigate", async () => {
    renderWithProviders(
      <GameOverModal game={finishedGame({ result: "white_wins", resultReason: "checkmate" })} />,
    );

    await userEvent.click(await screen.findByRole("button", { name: "Analyze game" }));

    expect(navigate).not.toHaveBeenCalled();
  });
});
