import { useCallback, useEffect, useState } from "react";
import {
  listBlocked,
  listFriends,
  listRequests,
  type BlockedUser,
  type Friend,
  type FriendRequest,
} from "./api";

export interface FriendsData {
  friends: Friend[];
  incoming: FriendRequest[];
  outgoing: FriendRequest[];
  blocked: BlockedUser[];
}

const EMPTY: FriendsData = { friends: [], incoming: [], outgoing: [], blocked: [] };

/**
 * Loads friends, requests and blocks together. REST-only for now: nothing
 * is pushed, so the lists are as fresh as the last load or `run`.
 */
export function useFriends() {
  const [data, setData] = useState<FriendsData>(EMPTY);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const [friends, requests, blocked] = await Promise.all([
        listFriends(),
        listRequests(),
        listBlocked(),
      ]);
      setData({ friends, ...requests, blocked });
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "could not load friends");
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- the initial fetch
    void refresh();
  }, [refresh]);

  /**
   * Runs a mutation, then reloads. Resolves to whether it succeeded; a
   * failure surfaces through `error` rather than throwing at the caller.
   */
  const run = useCallback(
    async (action: () => Promise<unknown>): Promise<boolean> => {
      try {
        await action();
      } catch (err) {
        setError(err instanceof Error ? err.message : "something went wrong");
        return false;
      }
      await refresh();
      return true;
    },
    [refresh],
  );

  return { ...data, isLoading, error, run };
}
