import { z } from "zod";

export const gameSchema = z.object({
  id: z.string(),
  whiteId: z.string(),
  blackId: z.string(),
  fen: z.string(),
  status: z.enum(["active", "finished"]),
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

export const opponentSchema = z.object({
  username: z.string(),
  rating: z.number(),
});

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
  | { type: "matchmaking.leave"; payload: { preset: string } };
