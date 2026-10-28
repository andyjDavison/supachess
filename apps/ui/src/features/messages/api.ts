import { z } from "zod";
import {
  conversationSchema,
  directMessageSchema,
  gameMessageSchema,
  type Conversation,
  type DirectMessage,
  type GameMessage,
} from "../../lib/ws/schemas";

const API_URL = import.meta.env.VITE_API_URL;

// The read side of chat (api/internal/transport/web/chat_handler.go).
// Sending is a WebSocket message - see store.ts and game-chat.tsx.

const conversationsResponseSchema = z.object({ conversations: z.array(conversationSchema) });
const messagesResponseSchema = z.object({ messages: z.array(directMessageSchema) });
const gameMessagesResponseSchema = z.object({ messages: z.array(gameMessageSchema) });
const userSchema = z.object({ id: z.string(), username: z.string(), rating: z.number() });

/** Matches the server's page size, so a short page means "no older messages". */
export const MESSAGE_PAGE_SIZE = 50;

async function request(path: string, init: RequestInit = {}): Promise<Response> {
  const res = await fetch(`${API_URL}${path}`, { credentials: "include", ...init });
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(text.trim() || `request failed (${res.status})`);
  }
  return res;
}

const id = (value: string) => encodeURIComponent(value);

export async function listConversations(): Promise<Conversation[]> {
  const res = await request("/api/conversations");
  return conversationsResponseSchema.parse(await res.json()).conversations;
}

/** One page, oldest first. `before` pages backwards from a message id. */
export async function listMessages(conversationId: string, before?: number): Promise<DirectMessage[]> {
  const query = before ? `?before=${before}` : "";
  const res = await request(`/api/conversations/${id(conversationId)}/messages${query}`);
  return messagesResponseSchema.parse(await res.json()).messages;
}

export async function markRead(conversationId: string) {
  await request(`/api/conversations/${id(conversationId)}/read`, { method: "POST" });
}

export async function archiveConversation(conversationId: string) {
  await request(`/api/conversations/${id(conversationId)}/archive`, { method: "PUT" });
}

export async function unarchiveConversation(conversationId: string) {
  await request(`/api/conversations/${id(conversationId)}/archive`, { method: "DELETE" });
}

export async function listGameMessages(gameId: string): Promise<GameMessage[]> {
  const res = await request(`/api/games/${id(gameId)}/chat`);
  return gameMessagesResponseSchema.parse(await res.json()).messages;
}

/** Public profile lookup, used to title a thread that doesn't exist yet. */
export async function getUser(userId: string) {
  const res = await request(`/api/user/${id(userId)}`);
  return userSchema.parse(await res.json());
}
