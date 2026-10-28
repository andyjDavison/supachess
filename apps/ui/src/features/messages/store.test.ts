import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FakeSocket, fakeSocketModule } from "../../test/fake-socket";
import { gameSocket } from "../../lib/ws/client";
import type { Conversation, DirectMessage } from "../../lib/ws/schemas";
import { selectUnseenCount, useMessagesStore } from "./store";

vi.mock("partysocket", () => fakeSocketModule());

const alice = { userId: "u-alice", username: "alice", rating: 1500 };
const ME = "u-me";

function dm(id: number, senderId: string, conversationId = "conv-1"): DirectMessage {
  return { id, conversationId, senderId, body: `msg ${id}`, sentAt: 1_700_000_000_000 + id };
}

function conv(overrides: Partial<Conversation> = {}): Conversation {
  return { id: "conv-1", other: alice, folder: "unread", unread: true, lastMessage: dm(1, alice.userId), ...overrides };
}

type Route = { method?: string; path: RegExp; status?: number; body?: unknown };

/** Fake REST backend: first matching route wins; records every call. */
function stubApi(routes: () => Route[]) {
  const fetchMock = vi.fn(async (url: string, init: RequestInit = {}) => {
    const method = init.method ?? "GET";
    const route = routes().find((r) => (r.method ?? "GET") === method && r.path.test(url));
    if (!route) throw new Error(`unexpected ${method} ${url}`);
    const status = route.status ?? 200;
    return { ok: status < 400, status, json: async () => route.body, text: async () => "" };
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

const calls = (fetchMock: ReturnType<typeof stubApi>, method: string, path: RegExp) =>
  fetchMock.mock.calls.filter(([url, init]) => (init?.method ?? "GET") === method && path.test(url as string));

beforeEach(() => {
  FakeSocket.reset();
  useMessagesStore.getState().reset();
  gameSocket.connect();
});

afterEach(() => {
  gameSocket.disconnect();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

const socket = () => FakeSocket.latest();
const state = () => useMessagesStore.getState();

function push(conversation: Conversation, message: DirectMessage) {
  socket().receive({ type: "chat.direct_message", payload: { conversation, message } });
}

describe("loading", () => {
  it("loads conversations newest first", async () => {
    stubApi(() => [
      {
        path: /\/api\/conversations$/,
        body: {
          conversations: [
            conv({ id: "old", lastMessage: dm(1, alice.userId, "old") }),
            conv({ id: "new", lastMessage: dm(9, alice.userId, "new") }),
          ],
        },
      },
    ]);

    await state().loadConversations();

    expect(state().loaded).toBe(true);
    expect(state().conversations.map((c) => c.id)).toEqual(["new", "old"]);
  });

  // A live message that lands while the list request is in flight must not
  // be overwritten by the (older) fetched copy.
  it("keeps a live update newer than the fetch", async () => {
    stubApi(() => [{ path: /\/api\/conversations$/, body: { conversations: [conv({ lastMessage: dm(1, alice.userId) })] } }]);
    push(conv({ lastMessage: dm(5, alice.userId) }), dm(5, alice.userId));

    await state().loadConversations();

    expect(state().conversations[0].lastMessage?.id).toBe(5);
  });

  it("counts conversations with unseen messages for the badge", () => {
    push(conv({ id: "a", unread: true }), dm(1, alice.userId, "a"));
    push(conv({ id: "b", unread: false, folder: "inbox" }), dm(2, ME, "b"));
    push(conv({ id: "c", unread: true, folder: "inbox" }), dm(3, alice.userId, "c"));

    expect(selectUnseenCount(state())).toBe(2);
  });
});

describe("live messages", () => {
  it("adds a new conversation and keeps the list newest first", () => {
    push(conv({ id: "a", lastMessage: dm(1, alice.userId, "a") }), dm(1, alice.userId, "a"));
    push(conv({ id: "b", lastMessage: dm(2, alice.userId, "b") }), dm(2, alice.userId, "b"));
    push(conv({ id: "a", lastMessage: dm(3, alice.userId, "a") }), dm(3, alice.userId, "a"));

    expect(state().conversations.map((c) => c.id)).toEqual(["a", "b"]);
  });

  it("takes the server's folder for the conversation (e.g. un-archived)", () => {
    push(conv({ folder: "archived", unread: false }), dm(1, ME));

    push(conv({ folder: "inbox", lastMessage: dm(2, alice.userId) }), dm(2, alice.userId));

    expect(state().conversations[0].folder).toBe("inbox");
  });

  it("appends to an open thread and marks it read straight away", async () => {
    const fetchMock = stubApi(() => [
      { path: /\/conv-1\/messages$/, body: { messages: [dm(1, alice.userId)] } },
      { method: "POST", path: /\/conv-1\/read$/, status: 204 },
    ]);
    push(conv({ unread: false }), dm(1, alice.userId));
    await state().openConversation("conv-1");

    push(conv({ unread: true, lastMessage: dm(2, alice.userId) }), dm(2, alice.userId));

    expect(state().threads["conv-1"].messages.map((m) => m.id)).toEqual([1, 2]);
    expect(state().conversations[0].unread).toBe(false);
    expect(calls(fetchMock, "POST", /\/read$/)).toHaveLength(1);
  });

  it("leaves a conversation that isn't open unread", () => {
    push(conv({ unread: true }), dm(1, alice.userId));

    expect(state().conversations[0].unread).toBe(true);
  });

  it("dedupes a message that arrives twice", async () => {
    stubApi(() => [{ path: /\/messages$/, body: { messages: [dm(1, alice.userId)] } }]);
    push(conv({ unread: false }), dm(1, alice.userId));
    await state().openConversation("conv-1");

    push(conv({ unread: false }), dm(1, alice.userId));

    expect(state().threads["conv-1"].messages).toHaveLength(1);
  });
});

describe("threads", () => {
  it("opening an unread conversation loads its history and marks it read", async () => {
    const fetchMock = stubApi(() => [
      { path: /\/conv-1\/messages$/, body: { messages: [dm(1, alice.userId)] } },
      { method: "POST", path: /\/conv-1\/read$/, status: 204 },
    ]);
    push(conv({ unread: true }), dm(1, alice.userId));

    await state().openConversation("conv-1");

    expect(state().activeId).toBe("conv-1");
    expect(state().conversations[0].unread).toBe(false);
    expect(state().threads["conv-1"]).toMatchObject({ loading: false, hasMore: false });
    expect(calls(fetchMock, "POST", /\/read$/)).toHaveLength(1);
  });

  it("pages backwards from the oldest loaded message", async () => {
    const fullPage = Array.from({ length: 50 }, (_, i) => dm(100 + i, alice.userId));
    const fetchMock = stubApi(() => [
      { path: /\/messages\?before=100$/, body: { messages: [dm(98, ME), dm(99, alice.userId)] } },
      { path: /\/messages$/, body: { messages: fullPage } },
    ]);
    push(conv({ unread: false, lastMessage: dm(149, alice.userId) }), dm(149, alice.userId));
    await state().openConversation("conv-1");
    expect(state().threads["conv-1"].hasMore).toBe(true);

    await state().loadEarlier("conv-1");

    const thread = state().threads["conv-1"];
    expect(thread.messages[0].id).toBe(98);
    expect(thread.messages).toHaveLength(52);
    expect(thread.hasMore).toBe(false); // a short page means that's everything
    expect(calls(fetchMock, "GET", /before=100/)).toHaveLength(1);
  });
});

describe("sending", () => {
  it("sends chat.send_direct", () => {
    state().send("u-alice", "hello");

    expect(socket().sentEnvelopes().at(-1)).toEqual({
      type: "chat.send_direct",
      payload: { toUserId: "u-alice", body: "hello" },
    });
    expect(state().sending).toBe(true);
  });

  it("the server's echo of my own message ends the send", () => {
    state().send("u-alice", "hello");

    push(conv({ folder: "inbox", unread: false, lastMessage: dm(1, ME) }), dm(1, ME));

    expect(state().sending).toBe(false);
  });

  it.each([
    ["rate_limited", "You're sending messages too fast. Wait a moment."],
    ["user_not_found", "That player isn't available."],
    ["invalid_message_body", "Messages must be between 1 and 500 characters."],
  ])("a %s reply becomes a notice", (code, notice) => {
    state().send("u-alice", "hello");

    socket().receive({ type: "error", payload: { code, message: code, inReplyTo: "chat.send_direct" } });

    expect(state().notice).toBe(notice);
    expect(state().sending).toBe(false);
  });

  it("ignores errors in reply to other features", () => {
    socket().receive({ type: "error", payload: { code: "user_not_found", message: "x", inReplyTo: "invite.offer" } });

    expect(state().notice).toBeNull();
  });
});

describe("folders", () => {
  it("archives optimistically", async () => {
    stubApi(() => [{ method: "PUT", path: /\/conv-1\/archive$/, status: 204 }]);
    push(conv({ folder: "inbox" }), dm(1, ME));

    await state().archive("conv-1");

    expect(state().conversations[0].folder).toBe("archived");
  });

  it("rolls back an archive the server refuses", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    stubApi(() => [{ method: "PUT", path: /\/archive$/, status: 404 }]);
    push(conv({ folder: "inbox" }), dm(1, ME));

    await state().archive("conv-1");

    expect(state().conversations[0].folder).toBe("inbox");
    expect(state().notice).not.toBeNull();
  });

  // Inbox or Unread depends on whether I've replied - only the server knows.
  it("unarchiving refetches to learn the folder", async () => {
    stubApi(() => [
      { method: "DELETE", path: /\/conv-1\/archive$/, status: 204 },
      { path: /\/api\/conversations$/, body: { conversations: [conv({ folder: "unread" })] } },
    ]);
    push(conv({ folder: "archived" }), dm(1, alice.userId));

    await state().unarchive("conv-1");

    expect(state().conversations[0].folder).toBe("unread");
  });
});

it("reset clears everything (sign-out)", () => {
  push(conv(), dm(1, alice.userId));

  state().reset();

  expect(state().conversations).toEqual([]);
  expect(state().loaded).toBe(false);
});
