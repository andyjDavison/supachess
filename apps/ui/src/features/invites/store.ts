import { create } from "zustand";
import { gameSocket } from "../../lib/ws/client";
import type { InviteCloseReason } from "../../lib/ws/schemas";

/** A pending challenge, from this user's side: `user` is the other player. */
export interface Invite {
  gameId: string;
  user: { userId: string; username: string; rating: number };
  preset: string;
  expiresAt: number;
}

interface InviteState {
  /** The one challenge this user has sent, once the server confirms it. */
  outgoing: Invite | null;
  /** An offer sent but not yet confirmed by invite.sent or refused by an error. */
  pendingOffer: { toUserId: string; username: string } | null;
  incoming: Invite[];
  /** Set by invite.accepted; the toaster navigates there and clears it. */
  acceptedGameId: string | null;
  /** The incoming invite we've just tried to accept, so an error can drop it. */
  answering: string | null;
  /** One human-readable line about the last thing that happened. */
  notice: string | null;

  offer: (toUserId: string, username: string, preset: string) => void;
  accept: (gameId: string) => void;
  decline: (gameId: string) => void;
  cancel: () => void;
  clearAccepted: () => void;
  dismissNotice: () => void;
  reset: () => void;
}

const initialState = {
  outgoing: null,
  pendingOffer: null,
  incoming: [],
  acceptedGameId: null,
  answering: null,
  notice: null,
};

/**
 * What to tell this user when an invite ends. Returns null when they caused
 * it themselves (cancelling or declining), since they already know.
 */
function closedNotice(
  reason: InviteCloseReason,
  username: string,
  mine: boolean,
): string | null {
  switch (reason) {
    case "declined":
      return mine ? `${username} declined your challenge.` : null;
    case "cancelled":
      return mine ? null : `${username} cancelled their challenge.`;
    case "expired":
      return mine ? `Your challenge to ${username} expired.` : null;
    case "unavailable":
      return `${username} is no longer available.`;
  }
}

// Error codes the invite flow can produce. The server's error envelope has
// no request id, so these are attributed to whatever invite action is in
// flight; any other code belongs to the game store and is ignored here.
function errorNotice(code: string, pendingUsername: string | undefined): string | null {
  switch (code) {
    case "user_not_connected":
      return `${pendingUsername ?? "That player"} is offline.`;
    case "user_not_found":
      // Also what a block looks like, deliberately.
      return "That player isn't available.";
    case "already_in_game":
      return "You or your opponent is already in a game.";
    case "invite_already_pending":
      return "You already have a challenge waiting for an answer.";
    case "invite_not_found":
      return "That challenge is no longer available.";
    case "cannot_invite_self":
      return "You can't challenge yourself.";
    default:
      return null;
  }
}

export const useInviteStore = create<InviteState>((set, get) => {
  gameSocket.subscribe((message) => {
    switch (message.type) {
      case "invite.sent": {
        const { gameId, to, preset, expiresAt } = message.payload;
        set({ outgoing: { gameId, user: to, preset, expiresAt }, pendingOffer: null });
        break;
      }
      case "invite.received": {
        const { gameId, from, preset, expiresAt } = message.payload;
        set((state) => ({
          incoming: [
            ...state.incoming.filter((i) => i.gameId !== gameId),
            { gameId, user: from, preset, expiresAt },
          ],
        }));
        break;
      }
      case "invite.accepted": {
        const { gameId } = message.payload;
        set((state) => ({
          acceptedGameId: gameId,
          outgoing: state.outgoing?.gameId === gameId ? null : state.outgoing,
          incoming: state.incoming.filter((i) => i.gameId !== gameId),
          answering: null,
          notice: null,
        }));
        break;
      }
      case "invite.closed": {
        const { gameId, reason } = message.payload;
        const { outgoing, incoming } = get();
        if (outgoing?.gameId === gameId) {
          set({ outgoing: null, notice: closedNotice(reason, outgoing.user.username, true) });
          break;
        }
        const theirs = incoming.find((i) => i.gameId === gameId);
        if (theirs) {
          set((state) => ({
            incoming: state.incoming.filter((i) => i.gameId !== gameId),
            answering: state.answering === gameId ? null : state.answering,
            notice: closedNotice(reason, theirs.user.username, false) ?? state.notice,
          }));
        }
        break;
      }
      case "error": {
        const { pendingOffer, answering } = get();
        const notice = errorNotice(message.payload.code, pendingOffer?.username);
        if (!notice) break;
        set((state) => ({
          notice,
          pendingOffer: null,
          answering: null,
          incoming: answering
            ? state.incoming.filter((i) => i.gameId !== answering)
            : state.incoming,
        }));
        break;
      }
    }
  });

  return {
    ...initialState,

    offer: (toUserId, username, preset) => {
      set({ pendingOffer: { toUserId, username }, notice: null });
      gameSocket.send({ type: "invite.offer", payload: { toUserId, preset } });
    },
    accept: (gameId) => {
      set({ answering: gameId, notice: null });
      gameSocket.send({ type: "invite.accept", payload: { gameId } });
    },
    decline: (gameId) => {
      // Optimistic: the server's invite.closed would remove it anyway.
      set((state) => ({ incoming: state.incoming.filter((i) => i.gameId !== gameId) }));
      gameSocket.send({ type: "invite.decline", payload: { gameId } });
    },
    cancel: () => {
      const gameId = get().outgoing?.gameId;
      if (!gameId) return;
      gameSocket.send({ type: "invite.cancel", payload: { gameId } });
    },
    clearAccepted: () => set({ acceptedGameId: null }),
    dismissNotice: () => set({ notice: null }),
    reset: () => set(initialState),
  };
});
