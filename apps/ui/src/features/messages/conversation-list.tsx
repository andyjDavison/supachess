import type { Conversation, Folder } from "../../lib/ws/schemas";
import { FOLDERS } from "./folders";
import { formatMessageTime } from "./format-time";

/** Inbox / Unread / Archived tabs, each with its conversation count. */
export function FolderTabs({
  conversations,
  folder,
  onChange,
}: {
  conversations: Conversation[];
  folder: Folder;
  onChange: (folder: Folder) => void;
}) {
  return (
    <div role="tablist" aria-label="Folders" className="flex bg-options-header rounded-t-sm">
      {FOLDERS.map(({ key, label }) => {
        const count = conversations.filter((c) => c.folder === key).length;
        return (
          <button
            key={key}
            role="tab"
            aria-selected={folder === key}
            onClick={() => onChange(key)}
            className={`flex-1 px-2 py-3 text-xs hover:cursor-pointer hover:text-text ${
              folder === key ? "bg-nav-bg text-text" : "text-clock-disabled"
            }`}
          >
            {label}
            {count > 0 && <span className="ml-1 text-clock-disabled">({count})</span>}
          </button>
        );
      })}
    </div>
  );
}

const EMPTY_TEXT: Record<Folder, string> = {
  inbox: "No conversations yet.",
  unread: "No new message requests.",
  archived: "Nothing archived.",
};

/** One folder's conversations, newest first, with a dot on unseen ones. */
export function ConversationList({
  conversations,
  folder,
  activeId,
  onSelect,
}: {
  conversations: Conversation[];
  folder: Folder;
  activeId?: string | null;
  onSelect: (conversation: Conversation) => void;
}) {
  const shown = conversations.filter((c) => c.folder === folder);

  if (shown.length === 0) {
    return <p className="p-4 text-sm text-clock-disabled">{EMPTY_TEXT[folder]}</p>;
  }

  return (
    <ul aria-label="Conversations" className="flex flex-col">
      {shown.map((c) => {
        const last = c.lastMessage;
        const mine = last && last.senderId !== c.other.userId;
        return (
          <li key={c.id}>
            <button
              onClick={() => onSelect(c)}
              aria-current={c.id === activeId ? "true" : undefined}
              className={`flex w-full items-center gap-2 px-3 py-2 text-left hover:cursor-pointer hover:bg-options-button ${
                c.id === activeId ? "bg-options-button" : ""
              }`}
            >
              <span className="flex min-w-0 flex-1 flex-col">
                <span className="flex items-baseline gap-2">
                  <span className={`truncate text-sm ${c.unread ? "text-text" : "text-text/80"}`}>
                    {c.other.username}
                  </span>
                  {last && (
                    <span className="ml-auto shrink-0 text-[10px] text-clock-disabled">
                      {formatMessageTime(last.sentAt)}
                    </span>
                  )}
                </span>
                {last && (
                  <span className="truncate text-xs text-clock-disabled">
                    {mine ? "You: " : ""}
                    {last.body}
                  </span>
                )}
              </span>
              {c.unread && <span aria-label="New messages" className="size-2 shrink-0 rounded-full bg-accent" />}
            </button>
          </li>
        );
      })}
    </ul>
  );
}
