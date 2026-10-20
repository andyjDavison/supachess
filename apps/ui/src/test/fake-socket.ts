import { vi } from "vitest";

/**
 * A stand-in for partysocket's WebSocket, so lib/ws/client.ts and anything
 * built on it can be driven without a server.
 *
 * Register it with `vi.mock("partysocket", () => fakeSocketModule())` at the
 * top of a test file, then reach the live instance through
 * `FakeSocket.latest()` to push frames in or inspect what was sent.
 */
export class FakeSocket {
  static instances: FakeSocket[] = [];

  url: string;
  sent: string[] = [];
  closed = false;
  private listeners: Record<string, Array<(event: unknown) => void>> = {};

  constructor(url: string) {
    this.url = url;
    FakeSocket.instances.push(this);
  }

  addEventListener(type: string, fn: (event: unknown) => void) {
    (this.listeners[type] ??= []).push(fn);
  }

  send(data: string) {
    this.sent.push(data);
  }

  close() {
    this.closed = true;
    this.emit("close", {});
  }

  /** Pushes a frame to the client as if the server had sent it. */
  receive(data: unknown) {
    this.emit("message", { data: typeof data === "string" ? data : JSON.stringify(data) });
  }

  /**
   * Pushes `data` through untouched, so a test can deliver something that
   * isn't a string - a binary frame, say - which receive() would otherwise
   * stringify on its way in.
   */
  receiveRaw(data: unknown) {
    this.emit("message", { data });
  }

  private emit(type: string, event: unknown) {
    for (const fn of this.listeners[type] ?? []) fn(event);
  }

  /** The parsed envelopes this socket was asked to send. */
  sentEnvelopes() {
    return this.sent.map((raw) => JSON.parse(raw));
  }

  static latest() {
    const socket = FakeSocket.instances.at(-1);
    if (!socket) throw new Error("no FakeSocket has been constructed yet");
    return socket;
  }

  static reset() {
    FakeSocket.instances = [];
  }
}

export function fakeSocketModule() {
  return { WebSocket: FakeSocket };
}

/** Silences the console.error calls the client makes on bad input. */
export function silenceConsoleError() {
  return vi.spyOn(console, "error").mockImplementation(() => {});
}
