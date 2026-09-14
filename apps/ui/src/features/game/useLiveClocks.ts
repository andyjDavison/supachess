import { useEffect, useMemo, useState } from "react";
import { Chess } from "chess.js";
import type { Game } from "../../lib/ws/schemas";

// The server only sends updated clock values on each move - between
// moves, nothing pushes a tick. This hook fills that gap locally: it
// re-renders on an interval and computes "how much time has elapsed
// since the last server-confirmed state" purely from real wall-clock
// time, rather than counting down a local timer that could drift from
// the server's actual authoritative values.
export function useLiveClocks(game: Game) {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (game.status !== "active") return; // no need to tick a finished game
    const id = setInterval(() => setNow(Date.now()), 250);
    return () => clearInterval(id);
  }, [game.status]);

  // Whose clock is actually running right now - derived from the FEN,
  // same source of truth the server itself uses to determine turn.
  const turn = useMemo(() => new Chess(game.fen).turn(), [game.fen]);

  const elapsedSinceLastMove =
    game.status === "active" ? Math.max(0, now - game.lastMoveAt) : 0;

  const whiteMs =
    turn === "w" && game.status === "active"
      ? Math.max(0, game.whiteTimeRemainingMs - elapsedSinceLastMove)
      : game.whiteTimeRemainingMs;

  const blackMs =
    turn === "b" && game.status === "active"
      ? Math.max(0, game.blackTimeRemainingMs - elapsedSinceLastMove)
      : game.blackTimeRemainingMs;

  return { whiteMs, blackMs, activeColor: turn };
}
