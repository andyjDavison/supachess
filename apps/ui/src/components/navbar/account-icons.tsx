import { useCallback, useState } from "react";
import { Link } from "react-router";
import friends from "../../assets/friends.svg";
import { MessagesPanel } from "../../features/messages/messages-panel";
import { selectUnseenCount, useMessagesStore } from "../../features/messages/store";
import { BellIcon, EnvelopeIcon, GearIcon } from "./icons";

const iconClass =
  "relative flex size-8 items-center justify-center rounded-sm text-text/80 hover:cursor-pointer hover:bg-bg hover:text-text";
const inertClass =
  "flex size-8 items-center justify-center rounded-sm text-text/30 cursor-not-allowed";

/**
 * The row of account shortcuts under the username: friends, notifications,
 * settings, messages. Notifications and settings are placeholders until
 * their own steps land.
 */
export function AccountIcons() {
  const [panelOpen, setPanelOpen] = useState(false);
  const unseen = useMessagesStore(selectUnseenCount);
  const close = useCallback(() => setPanelOpen(false), []);

  return (
    <>
      <div className="flex w-9/10 justify-between">
        <Link to="/friends" aria-label="Friends" title="Friends" className={iconClass}>
          <img src={friends} alt="" className="size-5" />
        </Link>
        <button type="button" aria-label="Notifications" title="Notifications — coming soon" disabled className={inertClass}>
          <BellIcon />
        </button>
        <button type="button" aria-label="Settings" title="Settings — coming soon" disabled className={inertClass}>
          <GearIcon />
        </button>
        <button
          type="button"
          data-messages-toggle
          aria-label={unseen > 0 ? `Messages, ${unseen} unread` : "Messages"}
          aria-expanded={panelOpen}
          title="Messages"
          onClick={() => setPanelOpen((open) => !open)}
          className={`${iconClass} ${panelOpen ? "bg-bg text-text" : ""}`}
        >
          <EnvelopeIcon />
          {unseen > 0 && (
            <span
              aria-hidden="true"
              className="absolute -right-1 -top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-accent-strong px-1 text-[10px] text-white"
            >
              {unseen > 9 ? "9+" : unseen}
            </span>
          )}
        </button>
      </div>
      {panelOpen && <MessagesPanel onClose={close} />}
    </>
  );
}
