import { z } from "zod";

export const gameSchema = z.object({
  id: z.string(),
  whiteId: z.string(),
  blackId: z.string(),
  fen: z.string(),
  // "waiting": an invite that hasn't been accepted yet; no clock runs.
  status: z.enum(["waiting", "active", "finished"]),
  result: z.string().optional(),
  resultReason: z.string().optional(),
  whiteTimeRemainingMs: z.number(),
  blackTimeRemainingMs: z.number(),
  // The time control, e.g. 3+2 is 180000 / 2000.
  initialMs: z.number(),
  incrementMs: z.number(),
  lastMoveAt: z.number(),
  finishedAt: z.number().nullable().optional(),
});
export type Game = z.infer<typeof gameSchema>;

export const moveSchema = z.object({
  gameId: z.string(),
  ply: z.number(),
  color: z.enum(["white", "black"]),
  from: z.string(),
  to: z.string(),
  promotion: z.string().optional(),
  san: z.string(),
  fenAfter: z.string(),
  playedAt: z.number(),
});
export type Move = z.infer<typeof moveSchema>;

// GET /api/games/{id}/moves encodes a bare array of MoveDTO - not an
// object wrapper. The unused MoveHistoryResponse struct in
// game_handler.go suggests otherwise; the handler doesn't use it.
export const movesResponseSchema = z.array(moveSchema);

export const opponentSchema = z.object({
  username: z.string(),
  rating: z.number(),
});

export const inviteUserSchema = z.object({
  userId: z.string(),
  username: z.string(),
  rating: z.number(),
});

export const inviteCloseReasonSchema = z.enum([
  "declined",
  "cancelled",
  "expired",
  "unavailable",
]);
export type InviteCloseReason = z.infer<typeof inviteCloseReasonSchema>;

// Chat shapes. Also returned by the REST read endpoints
// (/api/conversations, /api/games/{id}/chat), so they're exported for those.
export const directMessageSchema = z.object({
  id: z.number(),
  conversationId: z.string(),
  senderId: z.string(),
  body: z.string(),
  sentAt: z.number(),
});
export type DirectMessage = z.infer<typeof directMessageSchema>;

export const folderSchema = z.enum(["inbox", "unread", "archived"]);
export type Folder = z.infer<typeof folderSchema>;

export const conversationSchema = z.object({
  id: z.string(),
  other: inviteUserSchema,
  folder: folderSchema,
  unread: z.boolean(),
  lastMessage: directMessageSchema.nullable(),
});
export type Conversation = z.infer<typeof conversationSchema>;

export const gameMessageSchema = z.object({
  id: z.number(),
  gameId: z.string(),
  senderId: z.string(),
  body: z.string(),
  sentAt: z.number(),
});
export type GameMessage = z.infer<typeof gameMessageSchema>;

export const serverMessageSchema = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("game.move_made"),
    payload: z.object({ game: gameSchema, move: moveSchema }),
  }),
  z.object({
    type: z.literal("game.game_over"),
    payload: z.object({ game: gameSchema }),
  }),
  z.object({
    type: z.literal("game.opponent_disconnected"),
    payload: z.object({ gameId: z.string() }),
  }),
  z.object({
    type: z.literal("game.opponent_reconnected"),
    payload: z.object({ gameId: z.string() }),
  }),
  z.object({
    type: z.literal("matchmaking.matched"),
    payload: z.object({ gameId: z.string(), opponent: opponentSchema }),
  }),
  z.object({
    type: z.literal("invite.sent"),
    payload: z.object({
      gameId: z.string(),
      to: inviteUserSchema,
      preset: z.string(),
      expiresAt: z.number(),
    }),
  }),
  z.object({
    type: z.literal("invite.received"),
    payload: z.object({
      gameId: z.string(),
      from: inviteUserSchema,
      preset: z.string(),
      expiresAt: z.number(),
    }),
  }),
  z.object({
    type: z.literal("invite.accepted"),
    payload: z.object({ gameId: z.string(), opponent: opponentSchema }),
  }),
  z.object({
    type: z.literal("invite.closed"),
    payload: z.object({ gameId: z.string(), reason: inviteCloseReasonSchema }),
  }),
  z.object({
    type: z.literal("error"),
    payload: z.object({
      code: z.string(),
      message: z.string(),
      // The `type` of the client message that caused it; absent only when
      // the message was too malformed to have one.
      inReplyTo: z.string().optional(),
    }),
  }),
  z.object({
    type: z.literal("chat.direct_message"),
    payload: z.object({ conversation: conversationSchema, message: directMessageSchema }),
  }),
  z.object({
    type: z.literal("chat.game_message"),
    payload: z.object({ gameId: z.string(), message: gameMessageSchema }),
  }),
]);

export type ServerMessage = z.infer<typeof serverMessageSchema>;

export type ClientMessage =
  | {
      type: "game.submit_move";
      payload: { gameId: string; from: string; to: string; promotion?: string };
    }
  | { type: "game.resign"; payload: { gameId: string } }
  | { type: "game.subscribe_game"; payload: { gameId: string } }
  | { type: "matchmaking.join"; payload: { preset: string } }
  | { type: "matchmaking.leave"; payload: { preset: string } }
  | { type: "invite.offer"; payload: { toUserId: string; preset: string } }
  | { type: "invite.accept"; payload: { gameId: string } }
  | { type: "invite.decline"; payload: { gameId: string } }
  | { type: "invite.cancel"; payload: { gameId: string } }
  | { type: "chat.send_direct"; payload: { toUserId: string; body: string } }
  | { type: "chat.send_game"; payload: { gameId: string; body: string } };
