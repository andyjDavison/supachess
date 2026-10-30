import { act, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FakeSocket, fakeSocketModule } from "../../test/fake-socket";
import { gameSocket } from "../../lib/ws/client";
import { useGameStore } from "../game/store";
import { renderWithProviders } from "../../test/render";
import PlayOnline from "./play-online";

vi.mock("partysocket", () => fakeSocketModule());

const me = { id: "u1", email: "a@example.com", username: "alice" };

beforeEach(() => {
  FakeSocket.reset();
  gameSocket.connect();
  useGameStore.setState({ matchmakingStatus: "idle", queuedPreset: null, matchedGameId: null });
  vi.stubGlobal("fetch", vi.fn(async () => ({ ok: true, status: 200, json: async () => me })));
});

afterEach(() => {
  vi.useRealTimers();
  gameSocket.disconnect();
  vi.unstubAllGlobals();
});

const sent = () => FakeSocket.latest().sentEnvelopes();

describe("PlayOnline", () => {
  it("joins the queue at the chosen time control", async () => {
    renderWithProviders(<PlayOnline />);
    await screen.findByText("alice");

    await userEvent.click(screen.getByRole("radio", { name: "3 + 2" }));
    expect(screen.getByRole("radio", { name: "3 + 2" })).toHaveAttribute("aria-checked", "true");
    await userEvent.click(screen.getByRole("button", { name: "Play" }));

    expect(sent()).toContainEqual({ type: "matchmaking.join", payload: { preset: "3+2" } });
  });

  it("shows the chosen time control on the preview clocks", async () => {
    renderWithProviders(<PlayOnline />);
    await screen.findByText("alice");

    await userEvent.click(screen.getByRole("radio", { name: "5 min" }));

    expect(screen.getAllByText("5:00")).toHaveLength(2);
  });

  it("searches inside the panel, keeping the board, and can cancel", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    useGameStore.setState({ matchmakingStatus: "queued", queuedPreset: "10+5" });
    renderWithProviders(<PlayOnline />);
    await screen.findByText("alice");

    const status = screen.getByRole("status", { name: "Matchmaking" });
    expect(status).toHaveTextContent("Searching for an opponent…");
    expect(status).toHaveTextContent("10 + 5 (Rapid)");
    expect(screen.getByTestId("player-bar-black")).toHaveTextContent("Searching…");
    expect(screen.getByTestId("player-bar-white")).toHaveTextContent("alice");

    act(() => vi.advanceTimersByTime(7_000));
    expect(screen.getByLabelText("Time searching")).toHaveTextContent("0:07");

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(sent()).toContainEqual({ type: "matchmaking.leave", payload: { preset: "10+5" } });
    expect(screen.queryByRole("status", { name: "Matchmaking" })).not.toBeInTheDocument();
  });

  it("keeps the other tabs usable while searching", async () => {
    useGameStore.setState({ matchmakingStatus: "queued", queuedPreset: "1+0" });
    renderWithProviders(<PlayOnline />);
    await screen.findByText("alice");

    await userEvent.click(screen.getByRole("tab", { name: "Players" }));

    expect(screen.getByRole("tab", { name: "Players" })).toHaveAttribute("aria-selected", "true");
    expect(screen.queryByRole("status", { name: "Matchmaking" })).not.toBeInTheDocument();
    expect(useGameStore.getState().matchmakingStatus).toBe("queued");
  });
});
