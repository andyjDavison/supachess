import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router";
import { FakeSocket, fakeSocketModule } from "../../test/fake-socket";
import { gameSocket } from "../../lib/ws/client";
import { useInviteStore } from "./store";
import { InviteToasts } from "./invite-toasts";
import { formatCountdown } from "./format-countdown";
import { ChallengeButton } from "./challenge-button";

vi.mock("partysocket", () => fakeSocketModule());

const alice = { userId: "u-alice", username: "alice", rating: 1500 };
const bob = { userId: "u-bob", username: "bob", rating: 1300 };
const NOW = 1_700_000_000_000;

beforeEach(() => {
  FakeSocket.reset();
  useInviteStore.getState().reset();
  gameSocket.connect();
});

afterEach(() => {
  gameSocket.disconnect();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

const socket = () => FakeSocket.latest();
const lastSent = () => socket().sentEnvelopes().at(-1);

function renderToasts() {
  return render(
    <MemoryRouter initialEntries={["/friends"]}>
      <Routes>
        <Route path="/friends" element={<InviteToasts />} />
        <Route path="/game/:gameId" element={<p>game page</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

function receive(message: unknown) {
  act(() => socket().receive(message));
}

describe("formatCountdown", () => {
  it.each([
    [60_000, "1:00"],
    [42_100, "0:43"],
    [999, "0:01"],
    [0, "0:00"],
    [-5_000, "0:00"],
  ])("%d ms left -> %s", (left, want) => {
    expect(formatCountdown(NOW + left, NOW)).toBe(want);
  });
});

describe("InviteToasts", () => {
  it("renders nothing with no invites", () => {
    renderToasts();

    expect(screen.queryByRole("region", { name: "Challenges" })).not.toBeInTheDocument();
  });

  it("shows an incoming challenge with a live countdown", () => {
    vi.useFakeTimers({ now: NOW });
    renderToasts();

    receive({ type: "invite.received", payload: { gameId: "game-1", from: alice, preset: "3+2", expiresAt: NOW + 42_000 } });

    const region = screen.getByRole("region", { name: "Challenges" });
    expect(within(region).getByText("alice")).toBeInTheDocument();
    expect(within(region).getByText(/3\+2 · expires in 0:42/)).toBeInTheDocument();

    act(() => vi.advanceTimersByTime(2_000));
    expect(within(region).getByText(/expires in 0:40/)).toBeInTheDocument();
  });

  it("Accept and Decline send the matching message", async () => {
    renderToasts();
    receive({ type: "invite.received", payload: { gameId: "game-1", from: alice, preset: "3+2", expiresAt: NOW } });
    receive({ type: "invite.received", payload: { gameId: "game-2", from: bob, preset: "1+0", expiresAt: NOW } });

    await userEvent.click(screen.getAllByRole("button", { name: "Accept" })[0]);
    expect(lastSent()).toEqual({ type: "invite.accept", payload: { gameId: "game-1" } });

    await userEvent.click(screen.getAllByRole("button", { name: "Decline" })[1]);
    expect(lastSent()).toEqual({ type: "invite.decline", payload: { gameId: "game-2" } });
  });

  it("navigates into the game and subscribes once the invite is accepted", () => {
    renderToasts();

    receive({ type: "invite.accepted", payload: { gameId: "game-1", opponent: { username: "alice", rating: 1500 } } });

    expect(screen.getByText("game page")).toBeInTheDocument();
    expect(socket().sentEnvelopes()).toContainEqual({ type: "game.subscribe_game", payload: { gameId: "game-1" } });
    expect(useInviteStore.getState().acceptedGameId).toBeNull();
  });

  it("shows the outgoing challenge with Cancel", async () => {
    renderToasts();
    act(() => useInviteStore.getState().offer(bob.userId, bob.username, "5+0"));

    // Before the server confirms, there's no game id to cancel yet.
    expect(screen.getByText(/Waiting for bob/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();

    receive({ type: "invite.sent", payload: { gameId: "game-9", to: bob, preset: "5+0", expiresAt: NOW } });
    await userEvent.click(screen.getByRole("button", { name: "Cancel" }));

    expect(lastSent()).toEqual({ type: "invite.cancel", payload: { gameId: "game-9" } });
  });

  it("shows and dismisses the notice for how a challenge ended", async () => {
    renderToasts();
    act(() => useInviteStore.getState().offer(bob.userId, bob.username, "5+0"));
    receive({ type: "invite.sent", payload: { gameId: "game-9", to: bob, preset: "5+0", expiresAt: NOW } });

    receive({ type: "invite.closed", payload: { gameId: "game-9", reason: "declined" } });

    expect(screen.getByRole("status")).toHaveTextContent("bob declined your challenge.");
    await userEvent.click(screen.getByRole("button", { name: "Dismiss" }));
    expect(screen.queryByRole("region", { name: "Challenges" })).not.toBeInTheDocument();
  });

  it("clears invite state when it unmounts (sign-out)", () => {
    const { unmount } = renderToasts();
    receive({ type: "invite.received", payload: { gameId: "game-1", from: alice, preset: "3+2", expiresAt: NOW } });

    unmount();

    expect(useInviteStore.getState().incoming).toEqual([]);
  });
});

describe("ChallengeButton", () => {
  it("offers the chosen preset to that player", async () => {
    render(<ChallengeButton userId="u-bob" username="bob" />);

    await userEvent.click(screen.getByRole("button", { name: "Challenge" }));
    await userEvent.click(within(screen.getByRole("menu", { name: "Challenge bob" })).getByRole("menuitem", { name: "3 + 2" }));

    expect(lastSent()).toEqual({ type: "invite.offer", payload: { toUserId: "u-bob", preset: "3+2" } });
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("offers all nine presets", async () => {
    render(<ChallengeButton userId="u-bob" username="bob" />);

    await userEvent.click(screen.getByRole("button", { name: "Challenge" }));

    expect(screen.getAllByRole("menuitem")).toHaveLength(9);
  });

  it("is disabled while a challenge is already out", () => {
    act(() => useInviteStore.getState().offer("u-carol", "carol", "1+0"));

    render(<ChallengeButton userId="u-bob" username="bob" />);

    expect(screen.getByRole("button", { name: "Challenge" })).toBeDisabled();
  });
});
