import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FakeSocket, fakeSocketModule } from "../../test/fake-socket";
import { gameSocket } from "../../lib/ws/client";
import { useInviteStore } from "./store";

vi.mock("partysocket", () => fakeSocketModule());

const alice = { userId: "u-alice", username: "alice", rating: 1500 };
const bob = { userId: "u-bob", username: "bob", rating: 1300 };

beforeEach(() => {
  FakeSocket.reset();
  useInviteStore.getState().reset();
  gameSocket.connect();
});

afterEach(() => {
  gameSocket.disconnect();
  vi.restoreAllMocks();
});

const socket = () => FakeSocket.latest();
const state = () => useInviteStore.getState();

function received(gameId = "game-1", from = alice) {
  socket().receive({
    type: "invite.received",
    payload: { gameId, from, preset: "3+2", expiresAt: 1_700_000_060_000 },
  });
}

function offerAndConfirm(gameId = "game-9") {
  state().offer(bob.userId, bob.username, "5+0");
  socket().receive({
    type: "invite.sent",
    payload: { gameId, to: bob, preset: "5+0", expiresAt: 1_700_000_060_000 },
  });
}

describe("outgoing", () => {
  it("offer sends invite.offer and is pending until invite.sent confirms", () => {
    state().offer(bob.userId, bob.username, "5+0");

    expect(socket().sentEnvelopes().at(-1)).toEqual({
      type: "invite.offer",
      payload: { toUserId: "u-bob", preset: "5+0" },
    });
    expect(state().pendingOffer).toEqual({ toUserId: "u-bob", username: "bob" });

    socket().receive({
      type: "invite.sent",
      payload: { gameId: "game-9", to: bob, preset: "5+0", expiresAt: 1_700_000_060_000 },
    });

    expect(state().pendingOffer).toBeNull();
    expect(state().outgoing).toEqual({ gameId: "game-9", user: bob, preset: "5+0", expiresAt: 1_700_000_060_000 });
  });

  it("an offer refused by the server becomes a notice", () => {
    state().offer(bob.userId, bob.username, "5+0");

    socket().receive({ type: "error", payload: { code: "user_not_connected", message: "user not connected" } });

    expect(state().pendingOffer).toBeNull();
    expect(state().notice).toBe("bob is offline.");
  });

  it("cancel sends invite.cancel for the outgoing game", () => {
    offerAndConfirm("game-9");

    state().cancel();

    expect(socket().sentEnvelopes().at(-1)).toEqual({ type: "invite.cancel", payload: { gameId: "game-9" } });
  });

  it.each([
    ["declined", "bob declined your challenge."],
    ["expired", "Your challenge to bob expired."],
    ["unavailable", "bob is no longer available."],
    ["cancelled", null], // we cancelled it ourselves
  ])("closed as %s clears it with notice %j", (reason, notice) => {
    offerAndConfirm("game-9");

    socket().receive({ type: "invite.closed", payload: { gameId: "game-9", reason } });

    expect(state().outgoing).toBeNull();
    expect(state().notice).toBe(notice);
  });
});

describe("incoming", () => {
  it("collects incoming invites, without duplicating a redelivered one", () => {
    received("game-1", alice);
    received("game-2", bob);
    received("game-1", alice);

    expect(state().incoming.map((i) => i.gameId)).toEqual(["game-2", "game-1"]);
    expect(state().incoming[1].user.username).toBe("alice");
  });

  it("accept sends invite.accept; invite.accepted hands off the game", () => {
    received("game-1");

    state().accept("game-1");
    expect(socket().sentEnvelopes().at(-1)).toEqual({ type: "invite.accept", payload: { gameId: "game-1" } });

    socket().receive({
      type: "invite.accepted",
      payload: { gameId: "game-1", opponent: { username: "alice", rating: 1500 } },
    });

    expect(state().acceptedGameId).toBe("game-1");
    expect(state().incoming).toEqual([]);
  });

  it("decline sends invite.decline and drops the card straight away", () => {
    received("game-1");

    state().decline("game-1");

    expect(socket().sentEnvelopes().at(-1)).toEqual({ type: "invite.decline", payload: { gameId: "game-1" } });
    expect(state().incoming).toEqual([]);
  });

  it("an accept the server refuses drops that invite with a notice", () => {
    received("game-1");
    received("game-2", bob);

    state().accept("game-1");
    socket().receive({ type: "error", payload: { code: "invite_not_found", message: "invite not found" } });

    expect(state().incoming.map((i) => i.gameId)).toEqual(["game-2"]);
    expect(state().notice).toBe("That challenge is no longer available.");
  });

  it("the inviter cancelling removes it and says so", () => {
    received("game-1", alice);

    socket().receive({ type: "invite.closed", payload: { gameId: "game-1", reason: "cancelled" } });

    expect(state().incoming).toEqual([]);
    expect(state().notice).toBe("alice cancelled their challenge.");
  });
});

it("the inviter also gets invite.accepted and clears its outgoing invite", () => {
  offerAndConfirm("game-9");

  socket().receive({
    type: "invite.accepted",
    payload: { gameId: "game-9", opponent: { username: "bob", rating: 1300 } },
  });

  expect(state().outgoing).toBeNull();
  expect(state().acceptedGameId).toBe("game-9");
});

it("ignores error codes that belong to the game, not invites", () => {
  socket().receive({ type: "error", payload: { code: "illegal_move", message: "illegal move" } });

  expect(state().notice).toBeNull();
});

it("ignores invite.closed for an invite it doesn't know", () => {
  socket().receive({ type: "invite.closed", payload: { gameId: "nope", reason: "expired" } });

  expect(state().notice).toBeNull();
});
