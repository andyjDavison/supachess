import { useEffect, useState } from "react";
import { getUser } from "../messages/api";

export type Player = { username: string; rating: number };

/**
 * Both players' public profiles. Their `rating` is the Blitz display
 * rating; prefer a game's own `ratings` snapshot where there is one.
 */
export function usePlayers(whiteId: string, blackId: string) {
  const [players, setPlayers] = useState<{ white: Player | null; black: Player | null }>({ white: null, black: null });
  useEffect(() => {
    let cancelled = false;
    Promise.all([getUser(whiteId), getUser(blackId)])
      .then(([white, black]) => !cancelled && setPlayers({ white, black }))
      .catch((err) => console.error("loading players", err));
    return () => {
      cancelled = true;
    };
  }, [whiteId, blackId]);
  return players;
}
