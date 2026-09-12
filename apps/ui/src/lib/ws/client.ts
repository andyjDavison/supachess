import { WebSocket as ReconnectingWebSocket } from "partysocket";
import {
  serverMessageSchema,
  type ServerMessage,
  type ClientMessage,
} from "./schemas.ts";

type Listener = (message: ServerMessage) => void;

// Not exported directly as a class instance — see the singleton export at
// the bottom. A single WS connection is shared app-wide (per-user, per
// our protocol design), not one per component that happens to care about
// game state.
class GameSocketClient {
  private socket: ReconnectingWebSocket | null = null;
  private listeners = new Set<Listener>();

  connect() {
    if (this.socket) return; // already connected/connecting

    const wsUrl = import.meta.env.VITE_WS_URL;
    this.socket = new ReconnectingWebSocket(`${wsUrl}/ws`);

    this.socket.addEventListener("message", (event) => {
      this.handleRawMessage(event.data);
    });

    this.socket.addEventListener("close", () => {
      // partysocket handles the actual reconnect attempt itself; this is
      // just where you'd hook in UI state ("reconnecting...") later.
    });
  }

  disconnect() {
    this.socket?.close();
    this.socket = null;
  }

  private handleRawMessage(data: unknown) {
    if (typeof data !== "string") return; // ignore binary frames, unused by this protocol

    let parsed: unknown;
    try {
      parsed = JSON.parse(data);
    } catch {
      console.error("received non-JSON WS message", data);
      return;
    }

    const result = serverMessageSchema.safeParse(parsed);
    if (!result.success) {
      // A shape mismatch here means the frontend schema and the backend's
      // actual output have drifted apart — exactly the risk the protocol
      // doc exists to prevent. Logged loudly on purpose, not swallowed.
      console.error(
        "received WS message that failed schema validation",
        parsed,
        result.error,
      );
      return;
    }

    for (const listener of this.listeners) {
      listener(result.data);
    }
  }

  // subscribe returns an unsubscribe function — the standard pattern for
  // cleaning up in a useEffect without a separate removeListener call.
  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  send(message: ClientMessage) {
    if (!this.socket) {
      console.error(
        "attempted to send a WS message before connecting",
        message,
      );
      return;
    }
    this.socket.send(JSON.stringify(message));
  }
}

// Singleton — the protocol is designed around one connection per logged-in
// user, not one per component. Components subscribe to this shared
// instance rather than each creating their own connection.
export const gameSocket = new GameSocketClient();
