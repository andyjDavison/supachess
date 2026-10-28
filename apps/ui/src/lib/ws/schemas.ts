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
    payload: z.object({ code: z.string(), message: z.string() }),
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
  | { type: "invite.cancel"; payload: { gameId: string } };
