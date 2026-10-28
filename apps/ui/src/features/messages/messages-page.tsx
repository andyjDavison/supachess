import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";
import type { Folder } from "../../lib/ws/schemas";
import { getUser } from "./api";
import { Composer } from "./composer";
import { ConversationList, FolderTabs } from "./conversation-list";
import { formatMessageTime } from "./format-time";
import { useMessagesStore } from "./store";

const buttonClass =
  "text-xs px-3 py-1.5 rounded-md text-text bg-linear-to-b from-gradient-start to-gradient-end hover:from-hover-gradient-start hover:to-hover-gradient-end hover:cursor-pointer";

/**
 * /messages/:conversationId? - and /messages?to=<userId> to start a
 * conversation that doesn't exist yet. The first send creates it, and the
 * page moves to its real URL once the server's echo arrives.
 */
export default function MessagesPage() {
  const { conversationId } = useParams<{ conversationId?: string }>();
  const [searchParams] = useSearchParams();
  const toUserId = conversationId ? null : searchParams.get("to");
  const navigate = useNavigate();

  const { conversations, loaded, threads, notice, openConversation, loadEarlier, send, archive, unarchive, dismissNotice } =
    useMessagesStore();

  const active = conversations.find((c) => c.id === conversationId) ?? null;
  const existingWithTarget = toUserId ? conversations.find((c) => c.other.userId === toUserId) : undefined;

  // A tab the user picked; otherwise show whichever folder the open
  // conversation is in, so arriving from the panel lands on it.
  const [folderChoice, setFolderChoice] = useState<Folder | null>(null);
  const folder = folderChoice ?? active?.folder ?? "inbox";

  useEffect(() => {
    void openConversation(conversationId ?? null);
    return () => void openConversation(null);
  }, [conversationId, openConversation]);

  // ?to= someone we already talk to (or just did, via the first send).
  useEffect(() => {
    if (existingWithTarget) navigate(`/messages/${existingWithTarget.id}`, { replace: true });
  }, [existingWithTarget, navigate]);

  return (
    <div className="flex h-full w-full max-w-5xl mx-auto gap-4 text-text">
      <aside className="flex w-72 shrink-0 flex-col overflow-hidden rounded-sm bg-nav-bg">
        <h1 className="px-4 py-3 text-lg">Messages</h1>
        <FolderTabs conversations={conversations} folder={folder} onChange={setFolderChoice} />
        <div className="flex-1 overflow-y-auto">
          {loaded ? (
            <ConversationList
              conversations={conversations}
              folder={folder}
              activeId={conversationId}
              onSelect={(c) => navigate(`/messages/${c.id}`)}
            />
          ) : (
            <p className="p-4 text-sm text-clock-disabled">Loading…</p>
          )}
        </div>
      </aside>

      <section className="flex min-w-0 flex-1 flex-col overflow-hidden rounded-sm bg-nav-bg">
        {notice && (
          <div role="alert" className="flex items-center gap-2 bg-red-900/40 px-4 py-2 text-sm text-red-200">
            <p className="flex-1">{notice}</p>
            <button aria-label="Dismiss" onClick={dismissNotice} className="hover:cursor-pointer">
              ×
            </button>
          </div>
        )}

        {active ? (
          <>
            <header className="flex items-center gap-3 bg-options-header px-4 py-3">
              <h2 className="flex-1 truncate text-sm">
                {active.other.username}
                <span className="ml-2 text-xs text-clock-disabled">({active.other.rating})</span>
              </h2>
              {active.folder === "archived" ? (
                <button className={buttonClass} onClick={() => unarchive(active.id)}>
                  Unarchive
                </button>
              ) : (
                <button className={buttonClass} onClick={() => archive(active.id)}>
                  Archive
                </button>
              )}
            </header>
            <MessageThread
              thread={threads[active.id]}
              otherUserId={active.other.userId}
              otherName={active.other.username}
              onLoadEarlier={() => loadEarlier(active.id)}
            />
            <Composer onSend={(body) => send(active.other.userId, body)} />
          </>
        ) : toUserId ? (
          <DraftThread toUserId={toUserId} onSend={(body) => send(toUserId, body)} />
        ) : (
          <p className="m-auto text-sm text-clock-disabled">
            {conversationId && loaded ? "Conversation not found." : "Select a conversation."}
          </p>
        )}
      </section>
    </div>
  );
}

function MessageThread({
  thread,
  otherUserId,
  otherName,
  onLoadEarlier,
}: {
  thread: ReturnType<typeof useMessagesStore.getState>["threads"][string] | undefined;
  otherUserId: string;
  otherName: string;
  onLoadEarlier: () => void;
}) {
  const scroller = useRef<HTMLDivElement>(null);
  const lastId = thread?.messages.at(-1)?.id;

  // Stick to the newest message when one arrives (not when older ones load).
  useLayoutEffect(() => {
    const el = scroller.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [lastId]);

  if (!thread || (thread.loading && thread.messages.length === 0)) {
    return <p className="m-auto text-sm text-clock-disabled">Loading…</p>;
  }

  return (
    <div ref={scroller} className="flex flex-1 flex-col gap-2 overflow-y-auto p-4">
      {thread.hasMore && (
        <button
          onClick={onLoadEarlier}
          disabled={thread.loading}
          className="self-center text-xs text-clock-disabled hover:cursor-pointer hover:text-text"
        >
          {thread.loading ? "Loading…" : "Load earlier messages"}
        </button>
      )}
      {thread.messages.map((m) => {
        const theirs = m.senderId === otherUserId;
        return (
          <div key={m.id} className={`flex flex-col max-w-[75%] ${theirs ? "self-start" : "self-end items-end"}`}>
            <p
              className={`whitespace-pre-wrap break-words rounded-lg px-3 py-2 text-sm ${
                theirs ? "bg-options-button" : "bg-accent-strong text-white"
              }`}
            >
              <span className="sr-only">{theirs ? `${otherName}: ` : "You: "}</span>
              {m.body}
            </p>
            <span className="mt-0.5 text-[10px] text-clock-disabled">{formatMessageTime(m.sentAt)}</span>
          </div>
        );
      })}
    </div>
  );
}

/** A conversation that doesn't exist yet: just a title and a composer. */
function DraftThread({ toUserId, onSend }: { toUserId: string; onSend: (body: string) => void }) {
  const [name, setName] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    getUser(toUserId)
      .then((u) => !cancelled && setName(u.username))
      .catch(() => !cancelled && setName(null));
    return () => {
      cancelled = true;
    };
  }, [toUserId]);

  return (
    <>
      <header className="bg-options-header px-4 py-3">
        <h2 className="text-sm">{name ? `New message to ${name}` : "New message"}</h2>
      </header>
      <p className="m-auto text-sm text-clock-disabled">Say hello.</p>
      <Composer onSend={onSend} />
    </>
  );
}
