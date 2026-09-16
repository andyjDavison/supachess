import { create } from "zustand";
import { gameSocket } from "../../lib/ws/client";
import type { Game, Move } from "../../lib/ws/schemas";

interface GameState {
  currentGame: Game | null;
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
  clearMatch: () => void;
}

export const useGameStore = create<GameState>((set, get) => {
  gameSocket.subscribe((message) => {
    switch (message.type) {
      case "game.move_made":
        set({
          currentGame: message.payload.game,
          lastMove: message.payload.move,
        });
        break;
      case "game.game_over":
        set({ currentGame: message.payload.game });
        break;
      case "matchmaking.matched":
        set({
          matchmakingStatus: "matched",
          matchedGameId: message.payload.gameId,
          opponent: message.payload.opponent,
        });
        break;
      case "error":
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
    lastMove: null,
    matchmakingStatus: "idle",
    matchedGameId: null,
    opponent: null,
    queuedPreset: null,

    joinQueue: (preset) => {
      set({ matchmakingStatus: "queued" });
      gameSocket.send({ type: "matchmaking.join", payload: { preset } });
    },
    leaveQueue: () => {
      const preset = get().queuedPreset;
      if (!preset) return;
      set({ matchmakingStatus: "idle" });
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
    clearMatch: () => {
      set({ matchmakingStatus: "idle", matchedGameId: null, opponent: null });
    },
  };
});
