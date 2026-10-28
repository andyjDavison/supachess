import { create } from "zustand";
import { gameSocket } from "../../lib/ws/client";
import type { Conversation, DirectMessage, Folder } from "../../lib/ws/schemas";
import {
  MESSAGE_PAGE_SIZE,
  archiveConversation,
  listConversations,
  listMessages,
  markRead,
  unarchiveConversation,
} from "./api";

export interface Thread {
  messages: DirectMessage[];
  /** False once a page comes back short: there's nothing older. */
  hasMore: boolean;
  loading: boolean;
}

interface MessagesState {
  conversations: Conversation[];
  loaded: boolean;
  threads: Record<string, Thread>;
  /** The conversation open on screen; incoming messages to it are read at once. */
  activeId: string | null;
  /** A send is in flight, so a chat.send_direct error belongs to it. */
  sending: boolean;
  notice: string | null;

  loadConversations: () => Promise<void>;
  openConversation: (id: string | null) => Promise<void>;
  loadEarlier: (id: string) => Promise<void>;
  send: (toUserId: string, body: string) => void;
  archive: (id: string) => Promise<void>;
  unarchive: (id: string) => Promise<void>;
  dismissNotice: () => void;
  reset: () => void;
}

const initialState = {
  conversations: [],
  loaded: false,
  threads: {},
  activeId: null,
  sending: false,
  notice: null,
};

const byRecency = (a: Conversation, b: Conversation) =>
  (b.lastMessage?.id ?? 0) - (a.lastMessage?.id ?? 0);

/** Replaces a conversation by id (or adds it), keeping newest-first order. */
function upsert(list: Conversation[], conv: Conversation): Conversation[] {
  return [...list.filter((c) => c.id !== conv.id), conv].sort(byRecency);
}

/**
 * Combines a fresh fetch with what's in memory. The server's copy wins,
 * unless a live message newer than the fetch has already updated ours.
 */
function mergeConversations(fetched: Conversation[], local: Conversation[]): Conversation[] {
  const merged = new Map(fetched.map((c) => [c.id, c]));
  for (const c of local) {
    const server = merged.get(c.id);
    if (!server || (c.lastMessage?.id ?? 0) > (server.lastMessage?.id ?? 0)) merged.set(c.id, c);
  }
  return [...merged.values()].sort(byRecency);
}

/** Merges messages by id, oldest first - a live message can race a page load. */
function mergeMessages(a: DirectMessage[], b: DirectMessage[]): DirectMessage[] {
  const byId = new Map<number, DirectMessage>();
  for (const m of [...a, ...b]) byId.set(m.id, m);
  return [...byId.values()].sort((x, y) => x.id - y.id);
}

function errorNotice(code: string): string {
  switch (code) {
    case "rate_limited":
      return "You're sending messages too fast. Wait a moment.";
    case "invalid_message_body":
      return "Messages must be between 1 and 500 characters.";
    case "user_not_found":
      // Also what a block looks like, deliberately.
      return "That player isn't available.";
    case "cannot_message_self":
      return "You can't message yourself.";
    default:
      return "Your message couldn't be sent.";
  }
}

export const useMessagesStore = create<MessagesState>((set, get) => {
  gameSocket.subscribe((message) => {
    switch (message.type) {
      case "chat.direct_message": {
        const { conversation, message: dm } = message.payload;
        const fromOther = dm.senderId === conversation.other.userId;
        // Open on screen: it's being read as it arrives.
        const readNow = get().activeId === conversation.id && fromOther;
        set((state) => {
          const thread = state.threads[conversation.id];
          return {
            conversations: upsert(state.conversations, readNow ? { ...conversation, unread: false } : conversation),
            threads: thread
              ? { ...state.threads, [conversation.id]: { ...thread, messages: mergeMessages(thread.messages, [dm]) } }
              : state.threads,
            sending: fromOther ? state.sending : false,
          };
        });
        if (readNow) {
          markRead(conversation.id).catch((err) => console.error("marking read", err));
        }
        break;
      }
      case "error":
        if (message.payload.inReplyTo !== "chat.send_direct") break;
        set({ sending: false, notice: errorNotice(message.payload.code) });
        break;
    }
  });

  return {
    ...initialState,

    loadConversations: async () => {
      try {
        const fetched = await listConversations();
        set((state) => ({ conversations: mergeConversations(fetched, state.conversations), loaded: true }));
      } catch (err) {
        console.error("loading conversations", err);
        set({ loaded: true });
      }
    },

    openConversation: async (id) => {
      set({ activeId: id, notice: null });
      if (!id) return;

      if (get().conversations.find((c) => c.id === id)?.unread) {
        set((state) => ({
          conversations: state.conversations.map((c) => (c.id === id ? { ...c, unread: false } : c)),
        }));
        markRead(id).catch((err) => console.error("marking read", err));
      }

      if (get().threads[id]) return;
      set((state) => ({ threads: { ...state.threads, [id]: { messages: [], hasMore: true, loading: true } } }));
      try {
        const page = await listMessages(id);
        set((state) => ({
          threads: {
            ...state.threads,
            [id]: {
              messages: mergeMessages(page, state.threads[id]?.messages ?? []),
              hasMore: page.length === MESSAGE_PAGE_SIZE,
              loading: false,
            },
          },
        }));
      } catch (err) {
        console.error("loading messages", err);
        set((state) => ({ threads: { ...state.threads, [id]: { messages: [], hasMore: false, loading: false } } }));
      }
    },

    loadEarlier: async (id) => {
      const thread = get().threads[id];
      if (!thread || thread.loading || !thread.hasMore || thread.messages.length === 0) return;
      set((state) => ({ threads: { ...state.threads, [id]: { ...thread, loading: true } } }));
      try {
        const page = await listMessages(id, thread.messages[0].id);
        set((state) => ({
          threads: {
            ...state.threads,
            [id]: {
              messages: mergeMessages(page, state.threads[id].messages),
              hasMore: page.length === MESSAGE_PAGE_SIZE,
              loading: false,
            },
          },
        }));
      } catch (err) {
        console.error("loading earlier messages", err);
        set((state) => ({ threads: { ...state.threads, [id]: { ...state.threads[id], loading: false } } }));
      }
    },

    send: (toUserId, body) => {
      set({ sending: true, notice: null });
      gameSocket.send({ type: "chat.send_direct", payload: { toUserId, body } });
    },

    archive: async (id) => setFolder(id, "archived", archiveConversation),
    unarchive: async (id) => setFolder(id, null, unarchiveConversation),

    dismissNotice: () => set({ notice: null }),
    reset: () => set(initialState),
  };

  /**
   * Moves a conversation optimistically, rolling back if the server
   * refuses. Unarchiving goes to Inbox or Unread depending on whether this
   * user has replied, which only the server knows - so it refetches.
   */
  async function setFolder(id: string, folder: Folder | null, call: (id: string) => Promise<void>) {
    const before = get().conversations;
    if (folder) {
      set({ conversations: before.map((c) => (c.id === id ? { ...c, folder } : c)) });
    }
    try {
      await call(id);
      if (!folder) await get().loadConversations();
    } catch (err) {
      console.error("changing folder", err);
      set({ conversations: before, notice: "That didn't work. Try again." });
    }
  }
});

/** Conversations with a message this user hasn't opened - the navbar badge. */
export const selectUnseenCount = (s: MessagesState) => s.conversations.filter((c) => c.unread).length;
