import { create } from "zustand";
import { gameSocket } from "../../lib/ws/client";
import type { Game, Move } from "../../lib/ws/schemas";

// mergeMoves combines two move lists, keyed by ply. Entries from `b` win
// on collision, and the result is always ply-ordered.
//
// Used for both seeding and appending so the two can't fight: a
// game.move_made can arrive while the initial REST fetch is still in
// flight, and a plain assignment on arrival would drop whichever landed
// first. Dedupe also makes a redelivered move (reconnect resync) replace
// rather than duplicate, since the server resends full state.
export function mergeMoves(a: Move[], b: Move[]): Move[] {
  const byPly = new Map<number, Move>();
  for (const move of a) byPly.set(move.ply, move);
  for (const move of b) byPly.set(move.ply, move);
  return [...byPly.values()].sort((x, y) => x.ply - y.ply);
}

interface GameState {
  currentGame: Game | null;
  moves: Move[];
  lastMove: Move | null;
  matchmakingStatus: "idle" | "queued" | "matched";
  matchedGameId: string | null;
  opponent: { username: string; rating: number } | null;
  queuedPreset: string | null;

  joinQueue: (preset: string) => void;
  leaveQueue: () => void;
  submitMove: (from: string, to: string, promotion?: string) => void;
  resign: () => void;
  subscribeToGame: (gameId: string) => void;
  setInitialGame: (game: Game) => void;
  setInitialMoves: (moves: Move[]) => void;
  clearMatch: () => void;
}

export const useGameStore = create<GameState>((set, get) => {
  gameSocket.subscribe((message) => {
    switch (message.type) {
      case "game.move_made":
        // Updater form, unlike the sets below - appending needs the
        // current list, not just a replacement value.
        set((state) => ({
          currentGame: message.payload.game,
          lastMove: message.payload.move,
          moves: mergeMoves(state.moves, [message.payload.move]),
        }));
        break;
      case "game.game_over":
        set({ currentGame: message.payload.game });
        break;
      case "matchmaking.matched":
        set({
          matchmakingStatus: "matched",
          matchedGameId: message.payload.gameId,
          opponent: message.payload.opponent,
          queuedPreset: null,
        });
        break;
      case "error":
        // A refused join (already in a game, unknown preset) means we
        // never entered the queue; don't leave the page searching.
        if (message.payload.inReplyTo === "matchmaking.join") {
          set({ matchmakingStatus: "idle", queuedPreset: null });
        }
        console.error(
          "server error:",
          message.payload.code,
          message.payload.message,
        );
        break;
    }
  });

  return {
    currentGame: null,
    moves: [],
    lastMove: null,
    matchmakingStatus: "idle",
    matchedGameId: null,
    opponent: null,
    queuedPreset: null,

    joinQueue: (preset) => {
      set({ matchmakingStatus: "queued", queuedPreset: preset });
      gameSocket.send({ type: "matchmaking.join", payload: { preset } });
    },
    leaveQueue: () => {
      const preset = get().queuedPreset;
      if (!preset) return;
      set({ matchmakingStatus: "idle", queuedPreset: null });
      gameSocket.send({ type: "matchmaking.leave", payload: { preset } });
    },
    submitMove: (from, to, promotion) => {
      const gameId = get().currentGame?.id;
      if (!gameId) return;
      gameSocket.send({
        type: "game.submit_move",
        payload: { gameId, from, to, promotion },
      });
    },
    resign: () => {
      const gameId = get().currentGame?.id;
      if (!gameId) return;
      gameSocket.send({ type: "game.resign", payload: { gameId } });
    },
    subscribeToGame: (gameId) => {
      gameSocket.send({ type: "game.subscribe_game", payload: { gameId } });
    },
    setInitialGame: (game: Game) => set({ currentGame: game }),
    // Fetched moves go first so any that arrived live while the request
    // was in flight win - see mergeMoves.
    setInitialMoves: (moves: Move[]) =>
      set((state) => ({ moves: mergeMoves(moves, state.moves) })),
    clearMatch: () => {
      set({
        matchmakingStatus: "idle",
        matchedGameId: null,
        opponent: null,
        queuedPreset: null,
        moves: [],
      });
    },
  };
});
