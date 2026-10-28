import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { useGameStore } from "../game/store";
import { useInviteStore } from "./store";
import { formatCountdown } from "./format-countdown";

const buttonClass =
  "text-xs px-3 py-1.5 rounded-md text-text bg-linear-to-b from-gradient-start to-gradient-end hover:from-hover-gradient-start hover:to-hover-gradient-end hover:cursor-pointer";
const accentButtonClass =
  "text-xs px-3 py-1.5 rounded-md text-white bg-accent-strong hover:bg-accent hover:cursor-pointer";

/** Wall-clock millis, re-rendering every second while `active`. */
function useNow(active: boolean) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!active) return;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [active]);
  return now;
}

/**
 * App-wide challenge UI: incoming invites with Accept / Decline, the
 * outgoing one with Cancel, and a one-line notice for how the last one
 * ended. Mounted in App while a user is signed in, so it shows on any page.
 */
export function InviteToasts() {
  const { outgoing, pendingOffer, incoming, acceptedGameId, notice } = useInviteStore();
  const { accept, decline, cancel, clearAccepted, dismissNotice, reset } = useInviteStore();
  const subscribeToGame = useGameStore((s) => s.subscribeToGame);
  const navigate = useNavigate();
  const now = useNow(incoming.length > 0 || outgoing !== null);

  // Same hand-off as play-online.tsx does for matchmaking.matched.
  useEffect(() => {
    if (!acceptedGameId) return;
    subscribeToGame(acceptedGameId);
    navigate(`/game/${acceptedGameId}`);
    clearAccepted();
  }, [acceptedGameId, subscribeToGame, navigate, clearAccepted]);

  // Signing out unmounts this; don't carry one user's invites to the next.
  useEffect(() => reset, [reset]);

  if (incoming.length === 0 && !outgoing && !pendingOffer && !notice) return null;

  return (
    <div
      aria-label="Challenges"
      role="region"
      className="fixed bottom-4 right-4 z-50 flex w-80 max-w-[calc(100vw-2rem)] flex-col gap-2 text-text"
    >
      {incoming.map((inv) => (
        <div key={inv.gameId} className="rounded-md bg-nav-bg p-3 shadow-lg">
          <p className="text-sm">
            <span className="text-accent">{inv.user.username}</span> ({inv.user.rating}) challenges you
          </p>
          <p className="text-xs text-clock-disabled">
            {inv.preset} · expires in {formatCountdown(inv.expiresAt, now)}
          </p>
          <div className="mt-2 flex gap-2">
            <button className={accentButtonClass} onClick={() => accept(inv.gameId)}>
              Accept
            </button>
            <button className={buttonClass} onClick={() => decline(inv.gameId)}>
              Decline
            </button>
          </div>
        </div>
      ))}

      {(outgoing || pendingOffer) && (
        <div className="flex items-center gap-2 rounded-md bg-nav-bg p-3 shadow-lg">
          <p className="flex-1 text-sm">
            Waiting for {outgoing?.user.username ?? pendingOffer?.username}…
            {outgoing && (
              <span className="ml-1 text-xs text-clock-disabled">
                {outgoing.preset} · {formatCountdown(outgoing.expiresAt, now)}
              </span>
            )}
          </p>
          <button className={buttonClass} onClick={cancel} disabled={!outgoing}>
            Cancel
          </button>
        </div>
      )}

      {notice && (
        <div role="status" className="flex items-center gap-2 rounded-md bg-options-header p-3 text-sm shadow-lg">
          <p className="flex-1">{notice}</p>
          <button aria-label="Dismiss" className="text-clock-disabled hover:text-text hover:cursor-pointer" onClick={dismissNotice}>
            ×
          </button>
        </div>
      )}
    </div>
  );
}
