import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, useLocation } from "react-router";
import { FakeSocket, fakeSocketModule } from "../../test/fake-socket";
import { gameSocket } from "../../lib/ws/client";
import type { Conversation } from "../../lib/ws/schemas";
import { useMessagesStore } from "../../features/messages/store";
import { AccountIcons } from "./account-icons";

vi.mock("partysocket", () => fakeSocketModule());

function conversation(id: string, username: string, overrides: Partial<Conversation> = {}): Conversation {
  return {
    id,
    other: { userId: `u-${username}`, username, rating: 1200 },
    folder: "inbox",
    unread: false,
    lastMessage: { id: id.length, conversationId: id, senderId: `u-${username}`, body: `from ${username}`, sentAt: 1_700_000_000_000 },
    ...overrides,
  };
}

function Location() {
  return <p data-testid="location">{useLocation().pathname}</p>;
}

function renderIcons() {
  return render(
    <MemoryRouter initialEntries={["/play"]}>
      <AccountIcons />
      <p>outside</p>
      <Location />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  FakeSocket.reset();
  gameSocket.connect();
  useMessagesStore.setState({
    ...useMessagesStore.getInitialState(),
    loaded: true,
    conversations: [
      conversation("conv-a", "alice", { unread: true }),
      conversation("conv-bb", "bob", { folder: "unread", unread: true }),
      conversation("conv-ccc", "carol"),
    ],
  });
});

afterEach(() => {
  gameSocket.disconnect();
});

describe("AccountIcons", () => {
  it("links friends, and shows notifications and settings as not yet available", () => {
    renderIcons();

    expect(screen.getByRole("link", { name: "Friends" })).toHaveAttribute("href", "/friends");
    expect(screen.getByRole("button", { name: "Notifications" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Settings" })).toBeDisabled();
  });

  it("badges the envelope with the number of conversations with unseen messages", () => {
    renderIcons();

    expect(screen.getByRole("button", { name: "Messages, 2 unread" })).toHaveTextContent("2");
  });

  it("shows no badge when everything is read", () => {
    useMessagesStore.setState({ conversations: [conversation("conv-a", "alice")] });
    renderIcons();

    expect(screen.getByRole("button", { name: "Messages" })).not.toHaveTextContent(/\d/);
  });

  it("the envelope toggles the panel, which lists one folder at a time", async () => {
    renderIcons();
    const envelope = screen.getByRole("button", { name: /Messages/ });

    await userEvent.click(envelope);
    const panel = screen.getByRole("dialog", { name: "Messages" });
    expect(within(panel).getByText("alice")).toBeInTheDocument();
    expect(within(panel).queryByText("bob")).not.toBeInTheDocument();

    await userEvent.click(within(panel).getByRole("tab", { name: /Unread/ }));
    expect(within(panel).getByText("bob")).toBeInTheDocument();

    await userEvent.click(envelope);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("choosing a conversation opens it on /messages and closes the panel", async () => {
    renderIcons();
    await userEvent.click(screen.getByRole("button", { name: /Messages/ }));

    await userEvent.click(within(screen.getByRole("dialog")).getByText("carol"));

    expect(screen.getByTestId("location")).toHaveTextContent("/messages/conv-ccc");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("closes on Escape and on a click outside", async () => {
    renderIcons();
    const envelope = screen.getByRole("button", { name: /Messages/ });

    await userEvent.click(envelope);
    await userEvent.keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();

    await userEvent.click(envelope);
    await userEvent.click(screen.getByText("outside"));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
