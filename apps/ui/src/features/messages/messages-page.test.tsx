import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { FakeSocket, fakeSocketModule } from "../../test/fake-socket";
import { gameSocket } from "../../lib/ws/client";
import type { Conversation, DirectMessage } from "../../lib/ws/schemas";
import { useMessagesStore } from "./store";
import MessagesPage from "./messages-page";
import { Composer } from "./composer";
import { formatMessageTime } from "./format-time";

vi.mock("partysocket", () => fakeSocketModule());

const alice = { userId: "u-alice", username: "alice", rating: 1500 };
const bob = { userId: "u-bob", username: "bob", rating: 1300 };
const ME = "u-me";

function dm(id: number, senderId: string, conversationId: string, body = `msg ${id}`): DirectMessage {
  return { id, conversationId, senderId, body, sentAt: 1_700_000_000_000 + id };
}

const aliceConv: Conversation = {
  id: "conv-a",
  other: alice,
  folder: "inbox",
  unread: false,
  lastMessage: dm(2, alice.userId, "conv-a", "see you"),
};
const bobConv: Conversation = {
  id: "conv-b",
  other: bob,
  folder: "unread",
  unread: true,
  lastMessage: dm(3, bob.userId, "conv-b", "hi there"),
};

function stubApi() {
  const fetchMock = vi.fn(async (url: string, init: RequestInit = {}) => {
    const method = init.method ?? "GET";
    let body: unknown = null;
    let status = 200;
    if (/\/conv-a\/messages/.test(url)) body = { messages: [dm(1, ME, "conv-a", "hey alice"), dm(2, alice.userId, "conv-a", "see you")] };
    else if (/\/conv-b\/messages/.test(url)) body = { messages: [dm(3, bob.userId, "conv-b", "hi there")] };
    else if (/\/api\/user\/u-carol/.test(url)) body = { id: "u-carol", username: "carol", rating: 1100 };
    else if (method !== "GET") status = 204;
    else throw new Error(`unexpected ${method} ${url}`);
    return { ok: true, status, json: async () => body, text: async () => "" };
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function Location() {
  return <p data-testid="location">{useLocation().pathname + useLocation().search}</p>;
}

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/messages" element={<MessagesPage />} />
        <Route path="/messages/:conversationId" element={<MessagesPage />} />
      </Routes>
      <Location />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  FakeSocket.reset();
  gameSocket.connect();
  useMessagesStore.setState({ ...useMessagesStore.getInitialState(), conversations: [bobConv, aliceConv], loaded: true });
});

afterEach(() => {
  gameSocket.disconnect();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

const list = () => screen.getByRole("list", { name: "Conversations" });
const tab = (name: RegExp) => screen.getByRole("tab", { name });

describe("MessagesPage", () => {
  it("shows the Inbox folder by default, with per-folder counts", () => {
    stubApi();
    renderAt("/messages");

    expect(tab(/Inbox\s*\(1\)/)).toHaveAttribute("aria-selected", "true");
    expect(tab(/Unread\s*\(1\)/)).toBeInTheDocument();
    expect(within(list()).getByText("alice")).toBeInTheDocument();
    expect(within(list()).queryByText("bob")).not.toBeInTheDocument();
    expect(screen.getByText("Select a conversation.")).toBeInTheDocument();
  });

  it("switches folders, with a dot on unseen conversations", async () => {
    stubApi();
    renderAt("/messages");

    await userEvent.click(tab(/Unread/));

    expect(within(list()).getByText("bob")).toBeInTheDocument();
    expect(within(list()).getByLabelText("New messages")).toBeInTheDocument();
  });

  it("opens a conversation: thread, read receipt, and its folder selected", async () => {
    const fetchMock = stubApi();
    renderAt("/messages/conv-b");

    expect(await screen.findByText("hi there", { selector: "p" })).toBeInTheDocument();
    expect(tab(/Unread/)).toHaveAttribute("aria-selected", "true");
    expect(fetchMock).toHaveBeenCalledWith(expect.stringMatching(/\/conv-b\/read$/), expect.objectContaining({ method: "POST" }));
    expect(within(list()).queryByLabelText("New messages")).not.toBeInTheDocument();
  });

  it("clicking a conversation navigates to it", async () => {
    stubApi();
    renderAt("/messages");

    await userEvent.click(within(list()).getByText("alice"));

    expect(screen.getByTestId("location")).toHaveTextContent("/messages/conv-a");
    expect(await screen.findByText("hey alice")).toBeInTheDocument();
  });

  it("sends to the open conversation's other member", async () => {
    stubApi();
    renderAt("/messages/conv-a");
    await screen.findByText("hey alice");

    await userEvent.type(screen.getByLabelText("Message"), "how was your game?{Enter}");

    expect(FakeSocket.latest().sentEnvelopes().at(-1)).toEqual({
      type: "chat.send_direct",
      payload: { toUserId: "u-alice", body: "how was your game?" },
    });
    expect(screen.getByLabelText("Message")).toHaveValue("");
  });

  it("archives from the thread header", async () => {
    const fetchMock = stubApi();
    renderAt("/messages/conv-a");
    await screen.findByText("hey alice");

    await userEvent.click(screen.getByRole("button", { name: "Archive" }));

    expect(fetchMock).toHaveBeenCalledWith(expect.stringMatching(/\/conv-a\/archive$/), expect.objectContaining({ method: "PUT" }));
    expect(await screen.findByRole("button", { name: "Unarchive" })).toBeInTheDocument();
    expect(tab(/Archived\s*\(1\)/)).toHaveAttribute("aria-selected", "true");
  });

  it("?to= an existing conversation redirects to it", async () => {
    stubApi();
    renderAt("/messages?to=u-alice");

    await waitFor(() => expect(screen.getByTestId("location")).toHaveTextContent("/messages/conv-a"));
  });

  it("?to= someone new shows a draft, and moves to the conversation once it exists", async () => {
    stubApi();
    renderAt("/messages?to=u-carol");

    expect(await screen.findByText("New message to carol")).toBeInTheDocument();
    await userEvent.type(screen.getByLabelText("Message"), "hi carol{Enter}");
    expect(FakeSocket.latest().sentEnvelopes().at(-1)).toEqual({
      type: "chat.send_direct",
      payload: { toUserId: "u-carol", body: "hi carol" },
    });

    // The server's echo creates the conversation.
    act(() =>
      FakeSocket.latest().receive({
        type: "chat.direct_message",
        payload: {
          conversation: {
            id: "conv-c",
            other: { userId: "u-carol", username: "carol", rating: 1100 },
            folder: "inbox",
            unread: false,
            lastMessage: dm(4, ME, "conv-c", "hi carol"),
          },
          message: dm(4, ME, "conv-c", "hi carol"),
        },
      }),
    );

    await waitFor(() => expect(screen.getByTestId("location")).toHaveTextContent("/messages/conv-c"));
  });

  it("shows a send failure as a dismissable alert", async () => {
    stubApi();
    renderAt("/messages/conv-a");
    await screen.findByText("hey alice");
    await userEvent.type(screen.getByLabelText("Message"), "spam{Enter}");

    act(() =>
      FakeSocket.latest().receive({
        type: "error",
        payload: { code: "rate_limited", message: "slow down", inReplyTo: "chat.send_direct" },
      }),
    );

    expect(screen.getByRole("alert")).toHaveTextContent("too fast");
    await userEvent.click(screen.getByRole("button", { name: "Dismiss" }));
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("says so for a conversation that isn't yours", () => {
    stubApi();
    renderAt("/messages/conv-nope");

    expect(screen.getByText("Conversation not found.")).toBeInTheDocument();
  });
});

describe("Composer", () => {
  it("won't send blank or over-long messages", async () => {
    const onSend = vi.fn();
    render(<Composer onSend={onSend} />);
    const box = screen.getByLabelText("Message");

    await userEvent.type(box, "   {Enter}");
    expect(onSend).not.toHaveBeenCalled();

    await userEvent.clear(box);
    await userEvent.click(box);
    await userEvent.paste("x".repeat(501));
    expect(screen.getByText("501/500")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();
  });

  it("Shift+Enter is a newline, Enter sends the trimmed text", async () => {
    const onSend = vi.fn();
    render(<Composer onSend={onSend} />);

    await userEvent.type(screen.getByLabelText("Message"), " line one{Shift>}{Enter}{/Shift}line two {Enter}");

    expect(onSend).toHaveBeenCalledWith("line one\nline two");
  });

  it("counts an emoji as one character, like the server", async () => {
    render(<Composer onSend={vi.fn()} />);
    await userEvent.click(screen.getByLabelText("Message"));

    await userEvent.paste("😀".repeat(500));

    expect(screen.getByText("500/500")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send" })).toBeEnabled();
  });
});

describe("formatMessageTime", () => {
  const noon = new Date(2026, 9, 28, 12, 0).getTime();

  it("shows the time for today and the date otherwise", () => {
    expect(formatMessageTime(new Date(2026, 9, 28, 9, 5).getTime(), noon)).toMatch(/9:05/);
    expect(formatMessageTime(new Date(2026, 9, 20, 9, 5).getTime(), noon)).toMatch(/20/);
  });
});
