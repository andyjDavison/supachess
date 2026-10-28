import { useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router";
import type { Folder } from "../../lib/ws/schemas";
import { ConversationList, FolderTabs } from "./conversation-list";
import { useMessagesStore } from "./store";

/**
 * The quick-look panel that opens beside the navbar from the envelope
 * icon. List only: choosing a conversation opens it on /messages.
 */
export function MessagesPanel({ onClose }: { onClose: () => void }) {
  const conversations = useMessagesStore((s) => s.conversations);
  const loaded = useMessagesStore((s) => s.loaded);
  const [folder, setFolder] = useState<Folder>("inbox");
  const navigate = useNavigate();
  const panel = useRef<HTMLDivElement>(null);

  // Escape or a click outside closes it. The envelope button toggles it
  // itself, so clicks on the navbar's toggle are left to that handler.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    const onPointer = (e: PointerEvent) => {
      const target = e.target as Element | null;
      if (panel.current?.contains(target) || target?.closest("[data-messages-toggle]")) return;
      onClose();
    };
    document.addEventListener("keydown", onKey);
    document.addEventListener("pointerdown", onPointer);
    return () => {
      document.removeEventListener("keydown", onKey);
      document.removeEventListener("pointerdown", onPointer);
    };
  }, [onClose]);

  return (
    <div
      ref={panel}
      role="dialog"
      aria-label="Messages"
      className="fixed left-38 top-0 z-40 flex h-screen w-80 max-w-[calc(100vw-9.5rem)] flex-col bg-nav-sub-bg text-text shadow-xl"
    >
      <div className="flex items-center px-4 py-3">
        <h2 className="flex-1 text-sm">Messages</h2>
        <Link to="/messages" onClick={onClose} className="text-xs text-clock-disabled hover:text-text">
          Open all
        </Link>
      </div>
      <FolderTabs conversations={conversations} folder={folder} onChange={setFolder} />
      <div className="flex-1 overflow-y-auto bg-nav-bg">
        {loaded ? (
          <ConversationList
            conversations={conversations}
            folder={folder}
            onSelect={(c) => {
              navigate(`/messages/${c.id}`);
              onClose();
            }}
          />
        ) : (
          <p className="p-4 text-sm text-clock-disabled">Loading…</p>
        )}
      </div>
    </div>
  );
}
