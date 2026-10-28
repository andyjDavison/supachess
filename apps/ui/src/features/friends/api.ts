import { z } from "zod";

const API_URL = import.meta.env.VITE_API_URL;

// Mirrors the response structs in api/internal/transport/web/friends_handler.go.
// Timestamps are unix millis, like every other duration/time on the wire.

export const friendSchema = z.object({
  userId: z.string(),
  username: z.string(),
  rating: z.number(),
  sinceMs: z.number(),
});
export type Friend = z.infer<typeof friendSchema>;

export const friendRequestSchema = z.object({
  userId: z.string(),
  username: z.string(),
  rating: z.number(),
  createdAtMs: z.number(),
});
export type FriendRequest = z.infer<typeof friendRequestSchema>;

export const blockedUserSchema = z.object({
  userId: z.string(),
  username: z.string(),
  blockedAtMs: z.number(),
});
export type BlockedUser = z.infer<typeof blockedUserSchema>;

export const relationshipSchema = z.enum(["none", "friend", "outgoing", "incoming"]);
export type Relationship = z.infer<typeof relationshipSchema>;

export const searchResultSchema = z.object({
  userId: z.string(),
  username: z.string(),
  rating: z.number(),
  relationship: relationshipSchema,
});
export type SearchResult = z.infer<typeof searchResultSchema>;

const friendsListSchema = z.object({ friends: z.array(friendSchema) });
const requestsSchema = z.object({
  incoming: z.array(friendRequestSchema),
  outgoing: z.array(friendRequestSchema),
});
const blockedListSchema = z.object({ blocked: z.array(blockedUserSchema) });
const searchResponseSchema = z.object({ results: z.array(searchResultSchema) });
const sendResponseSchema = z.object({ status: z.enum(["pending", "accepted"]) });

/** A non-2xx response. `message` is the server's plain-text error body. */
export class FriendsApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "FriendsApiError";
    this.status = status;
  }
}

async function request(path: string, init: RequestInit = {}): Promise<Response> {
  const res = await fetch(`${API_URL}${path}`, { credentials: "include", ...init });
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new FriendsApiError(res.status, text.trim() || `request failed (${res.status})`);
  }
  return res;
}

async function getJson<T>(path: string, schema: z.ZodType<T>): Promise<T> {
  const res = await request(path);
  return schema.parse(await res.json());
}

const userPath = (id: string) => encodeURIComponent(id);

export async function listFriends(): Promise<Friend[]> {
  return (await getJson("/api/friends", friendsListSchema)).friends;
}

export function listRequests() {
  return getJson("/api/friends/requests", requestsSchema);
}

export async function listBlocked(): Promise<BlockedUser[]> {
  return (await getJson("/api/friends/blocks", blockedListSchema)).blocked;
}

export async function searchUsers(q: string): Promise<SearchResult[]> {
  return (await getJson(`/api/friends/search?q=${encodeURIComponent(q)}`, searchResponseSchema)).results;
}

/** Resolves to "accepted" when the other user had already asked us. */
export async function sendFriendRequest(userId: string) {
  const res = await request("/api/friends/requests", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ userId }),
  });
  return sendResponseSchema.parse(await res.json()).status;
}

export async function acceptFriendRequest(userId: string) {
  await request(`/api/friends/requests/${userPath(userId)}/accept`, { method: "POST" });
}

/** Declines a request we received, or cancels one we sent. */
export async function deleteFriendRequest(userId: string) {
  await request(`/api/friends/requests/${userPath(userId)}`, { method: "DELETE" });
}

export async function removeFriend(userId: string) {
  await request(`/api/friends/${userPath(userId)}`, { method: "DELETE" });
}

export async function blockUser(userId: string) {
  await request(`/api/friends/blocks/${userPath(userId)}`, { method: "PUT" });
}

export async function unblockUser(userId: string) {
  await request(`/api/friends/blocks/${userPath(userId)}`, { method: "DELETE" });
}
