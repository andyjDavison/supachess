import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { usePlayers } from "../game/use-players";
import { gameSocket } from "../../lib/ws/client";
import type { Game, GameMessage } from "../../lib/ws/schemas";
import { useAuth } from "../../stores/AuthContext";
import { listGameMessages } from "./api";
import { Composer } from "./composer";
import { describeGameEnd, formatStakes, formatTimeControl, ratingLine } from "./game-chat-text";

function errorNotice(code: string): string {
  switch (code) {
    case "rate_limited":
      return "You're sending messages too fast. Wait a moment.";
    case "game_already_finished":
      return "The game is over; chat is closed.";
    case "invalid_message_body":
      return "Messages must be between 1 and 500 characters.";
    default:
      return "Your message couldn't be sent.";
  }
}

/**
 * The Chat sub-tab of the game page's Analysis panel. It opens with a New
 * Game line and, once the game is over, closes with a Game Over line.
 * Only the two players can post, and only while the game is active; the
 * log stays readable afterwards. State is local: it's only ever shown for
 * one game, and refetching on mount is cheap.
 *
 * Ratings come from the game itself (its category's ratings at the start),
 * falling back to the profiles' for games from before ratings existed. A
 * player also sees their stakes under New Game and their new rating under
 * Game Over; spectators don't.
 */
export function GameChat({ game }: { game: Game }) {
  const { user } = useAuth();
  const [messages, setMessages] = useState<GameMessage[]>([]);
  const [notice, setNotice] = useState<string | null>(null);
  const scroller = useRef<HTMLDivElement>(null);
  const isPlayer = user?.id === game.whiteId || user?.id === game.blackId;
  const open = game.status === "active" && isPlayer;
  const { white, black } = usePlayers(game.whiteId, game.blackId);
  const whiteName = white?.username ?? "White";
  const blackName = black?.username ?? "Black";
  const timeControl = formatTimeControl(game.initialMs, game.incrementMs);
  const nameOf = (id: string) => (id === game.whiteId ? whiteName : id === game.blackId ? blackName : "?");
  const rated = (name: string, rating: number | undefined) => (rating === undefined ? name : `${name} (${rating})`);
  const mine = user?.id === game.whiteId ? game.ratings?.white : user?.id === game.blackId ? game.ratings?.black : undefined;

  useEffect(() => {
    let cancelled = false;
    const append = (incoming: GameMessage[]) =>
      setMessages((current) => {
        const byId = new Map(current.map((m) => [m.id, m]));
        for (const m of incoming) byId.set(m.id, m);
        return [...byId.values()].sort((a, b) => a.id - b.id);
      });

    // Subscribe before fetching, so a message that lands mid-request is
    // merged rather than lost.
    const unsubscribe = gameSocket.subscribe((msg) => {
      if (msg.type === "chat.game_message" && msg.payload.gameId === game.id) {
        append([msg.payload.message]);
      } else if (msg.type === "error" && msg.payload.inReplyTo === "chat.send_game") {
        setNotice(errorNotice(msg.payload.code));
      }
    });
    listGameMessages(game.id)
      .then((log) => !cancelled && append(log))
      .catch((err) => console.error("loading game chat", err));

    return () => {
      cancelled = true;
      unsubscribe();
    };
  }, [game.id]);

  const lastId = messages.at(-1)?.id;
  useLayoutEffect(() => {
    const el = scroller.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [lastId]);

  const send = (body: string) => {
    setNotice(null);
    gameSocket.send({ type: "chat.send_game", payload: { gameId: game.id, body } });
  };

  return (
    <div className="flex flex-1 min-h-0 flex-col rounded-b-sm bg-nav-bg text-text">
      <div ref={scroller} aria-label="Game chat" role="log" className="flex flex-1 flex-col gap-1 overflow-y-auto p-3">
        <SystemBlock title="New Game">
          <p>
            {rated(whiteName, game.ratings?.white.rating ?? white?.rating)} vs.{" "}
            {rated(blackName, game.ratings?.black.rating ?? black?.rating)} ({timeControl})
          </p>
          {mine && <p>{formatStakes(mine)}</p>}
        </SystemBlock>
        {messages.map((m) => (
          <p key={m.id} className="whitespace-pre-wrap break-words text-sm">
            <span className={m.senderId === user?.id ? "text-accent" : "text-clock-disabled"}>{nameOf(m.senderId)}:</span>{" "}
            {m.body}
          </p>
        ))}
        {game.status === "finished" && (
          <SystemBlock title="Game Over">
            <p>
              {describeGameEnd(game, whiteName, blackName)} ({timeControl})
            </p>
            {game.ratings && mine?.delta !== undefined && <p>{ratingLine(game.ratings.category, mine.rating, mine.delta)}</p>}
          </SystemBlock>
        )}
      </div>
      {notice && (
        <p role="alert" className="px-3 py-1 text-xs text-red-300">
          {notice}
        </p>
      )}
      {open ? (
        <Composer onSend={send} placeholder="Say something to your opponent…" />
      ) : (
        <p className="border-t border-options-header p-3 text-center text-xs text-clock-disabled">
          {isPlayer ? "Chat closed." : "Only the players can chat."}
        </p>
      )}
    </div>
  );
}

function SystemBlock({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="my-1 text-xs text-clock-disabled">
      <p className="text-sm text-text">{title}</p>
      {children}
    </div>
  );
}
