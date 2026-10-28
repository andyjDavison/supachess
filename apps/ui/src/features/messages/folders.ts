import type { Folder } from "../../lib/ws/schemas";

export const FOLDERS: { key: Folder; label: string }[] = [
  { key: "inbox", label: "Inbox" },
  { key: "unread", label: "Unread" },
  { key: "archived", label: "Archived" },
];
