import { useEffect } from "react";
import { useMessagesStore } from "./store";

/**
 * Loads the conversation list once per sign-in, so the navbar badge is
 * right on any page; the WebSocket keeps it current from then on. Mounted
 * in App while a user is signed in - unmounting on sign-out clears it, so
 * one user's messages never carry over to the next.
 */
export function MessagesBootstrap() {
  const loadConversations = useMessagesStore((s) => s.loadConversations);
  const reset = useMessagesStore((s) => s.reset);

  useEffect(() => {
    void loadConversations();
    return reset;
  }, [loadConversations, reset]);

  return null;
}
