import { useEffect, useState, type ReactNode } from "react";
import {
  acceptFriendRequest,
  blockUser,
  deleteFriendRequest,
  removeFriend,
  searchUsers,
  sendFriendRequest,
  unblockUser,
  type SearchResult,
} from "./api";
import { useFriends } from "./useFriends";
import { ChallengeButton } from "../invites/challenge-button";
import { MessageButton } from "./message-button";

const SEARCH_DEBOUNCE_MS = 250;

const buttonClass =
  "text-xs px-3 py-1.5 rounded-md text-text bg-linear-to-b from-gradient-start to-gradient-end hover:from-hover-gradient-start hover:to-hover-gradient-end hover:cursor-pointer disabled:opacity-50 disabled:cursor-default";
const accentButtonClass =
  "text-xs px-3 py-1.5 rounded-md text-white bg-accent-strong hover:bg-accent hover:cursor-pointer disabled:opacity-50 disabled:cursor-default";

export default function FriendsPage() {
  const { friends, incoming, outgoing, blocked, isLoading, error, run } = useFriends();
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  // Bumped after every mutation so the search results re-fetch and their
  // Add / Pending / Friends buttons reflect the new relationship.
  const [searchVersion, setSearchVersion] = useState(0);
  const [showBlocked, setShowBlocked] = useState(false);

  useEffect(() => {
    const q = query.trim();
    if (!q) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- clearing stale results
      setResults([]);
      return;
    }

    let cancelled = false;
    const timer = setTimeout(() => {
      searchUsers(q)
        .then((found) => {
          if (!cancelled) setResults(found);
        })
        .catch((err) => console.error("searching users", err));
    }, SEARCH_DEBOUNCE_MS);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [query, searchVersion]);

  const act = async (action: () => Promise<unknown>) => {
    await run(action);
    setSearchVersion((v) => v + 1);
  };

  const block = (userId: string, username: string) => {
    if (window.confirm(`Block ${username}? This also removes them as a friend.`)) {
      void act(() => blockUser(userId));
    }
  };

  return (
    <div className="flex flex-col w-full max-w-3xl mx-auto gap-4 p-4 text-text">
      <h1 className="text-2xl">Friends</h1>

      {error && (
        <p role="alert" className="rounded-md bg-red-900/40 px-4 py-2 text-sm text-red-200">
          {error}
        </p>
      )}

      <Section title="Find players">
        <input
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search by username"
          aria-label="Search by username"
          maxLength={32}
          className="w-full rounded-md border border-input-border bg-input-bg px-3 py-2 text-sm text-text placeholder:text-clock-disabled focus:outline-none focus:border-accent"
        />
        {query.trim() && results.length === 0 && (
          <p className="text-sm text-clock-disabled">No players found.</p>
        )}
        {results.map((r) => (
          <UserRow key={r.userId} username={r.username} rating={r.rating}>
            <SearchAction result={r} act={act} />
            <ChallengeButton userId={r.userId} username={r.username} />
            <MessageButton userId={r.userId} />
            <button className={buttonClass} onClick={() => block(r.userId, r.username)}>
              Block
            </button>
          </UserRow>
        ))}
      </Section>

      {isLoading ? (
        <p className="text-sm text-clock-disabled">Loading…</p>
      ) : (
        <>
          {incoming.length > 0 && (
            <Section title={`Requests (${incoming.length})`}>
              {incoming.map((r) => (
                <UserRow key={r.userId} username={r.username} rating={r.rating}>
                  <button className={accentButtonClass} onClick={() => act(() => acceptFriendRequest(r.userId))}>
                    Accept
                  </button>
                  <button className={buttonClass} onClick={() => act(() => deleteFriendRequest(r.userId))}>
                    Decline
                  </button>
                </UserRow>
              ))}
            </Section>
          )}

          {outgoing.length > 0 && (
            <Section title={`Sent (${outgoing.length})`}>
              {outgoing.map((r) => (
                <UserRow key={r.userId} username={r.username} rating={r.rating}>
                  <button className={buttonClass} onClick={() => act(() => deleteFriendRequest(r.userId))}>
                    Cancel
                  </button>
                </UserRow>
              ))}
            </Section>
          )}

          <Section title={`Friends (${friends.length})`}>
            {friends.length === 0 && (
              <p className="text-sm text-clock-disabled">No friends yet. Search for players above.</p>
            )}
            {friends.map((f) => (
              <UserRow key={f.userId} username={f.username} rating={f.rating}>
                <ChallengeButton userId={f.userId} username={f.username} />
                <MessageButton userId={f.userId} />
                <button
                  className={buttonClass}
                  onClick={() => {
                    if (window.confirm(`Remove ${f.username} from your friends?`)) {
                      void act(() => removeFriend(f.userId));
                    }
                  }}
                >
                  Remove
                </button>
                <button className={buttonClass} onClick={() => block(f.userId, f.username)}>
                  Block
                </button>
              </UserRow>
            ))}
          </Section>

          {blocked.length > 0 && (
            <div className="flex flex-col">
              <button
                className="self-start text-sm text-clock-disabled hover:text-text hover:cursor-pointer"
                aria-expanded={showBlocked}
                onClick={() => setShowBlocked((v) => !v)}
              >
                {showBlocked ? "▾" : "▸"} Blocked ({blocked.length})
              </button>
              {showBlocked && (
                <div className="mt-2 flex flex-col gap-2 rounded-sm bg-nav-bg p-4">
                  {blocked.map((b) => (
                    <UserRow key={b.userId} username={b.username}>
                      <button className={buttonClass} onClick={() => act(() => unblockUser(b.userId))}>
                        Unblock
                      </button>
                    </UserRow>
                  ))}
                </div>
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
}

function SearchAction({
  result,
  act,
}: {
  result: SearchResult;
  act: (action: () => Promise<unknown>) => Promise<void>;
}) {
  switch (result.relationship) {
    case "friend":
      return <span className="text-xs text-clock-disabled px-3">Friends</span>;
    case "outgoing":
      return <span className="text-xs text-clock-disabled px-3">Pending</span>;
    case "incoming":
      return (
        <button className={accentButtonClass} onClick={() => act(() => acceptFriendRequest(result.userId))}>
          Accept
        </button>
      );
    case "none":
      return (
        <button className={accentButtonClass} onClick={() => act(() => sendFriendRequest(result.userId))}>
          Add friend
        </button>
      );
  }
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section aria-label={title} className="flex flex-col">
      <h2 className="rounded-t-sm bg-options-header px-4 py-3 text-sm">{title}</h2>
      <div className="flex flex-col gap-2 rounded-b-sm bg-nav-bg p-4">{children}</div>
    </section>
  );
}

function UserRow({
  username,
  rating,
  children,
}: {
  username: string;
  rating?: number;
  children: ReactNode;
}) {
  return (
    <div className="flex items-center gap-3 rounded-md bg-options-button px-3 py-2">
      <span className="flex-1 min-w-0 truncate text-sm">
        {username}
        {rating !== undefined && <span className="ml-2 text-xs text-clock-disabled">({rating})</span>}
      </span>
      <div className="flex shrink-0 gap-2">{children}</div>
    </div>
  );
}
