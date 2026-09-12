import { useEffect } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../../stores/AuthContext";
import { useAuthModal } from "../../stores/AuthModalContext";
import { useGameStore } from "../game/store";

const PRESETS = [
  { key: "bullet", label: "Bullet · 1 min" },
  { key: "blitz", label: "Blitz · 5 | 3" },
  { key: "rapid", label: "Rapid · 10 min" },
];

export default function PlayOnline() {
  const { user } = useAuth();
  const { openModal } = useAuthModal();
  const navigate = useNavigate();

  const matchmakingStatus = useGameStore((s) => s.matchmakingStatus);
  const matchedGameId = useGameStore((s) => s.matchedGameId);
  const opponent = useGameStore((s) => s.opponent);
  const joinQueue = useGameStore((s) => s.joinQueue);
  const leaveQueue = useGameStore((s) => s.leaveQueue);
  const subscribeToGame = useGameStore((s) => s.subscribeToGame);
  const clearMatch = useGameStore((s) => s.clearMatch);

  // The actual "matched -> enter the game" handoff. Deliberately not
  // inside the store itself (per the earlier "navigation isn't the
  // store's job" decision) - this is the one place that reacts to it.
  useEffect(() => {
    if (!matchedGameId) return;

    subscribeToGame(matchedGameId);
    navigate(`/game/${matchedGameId}`);
    clearMatch();
  }, [matchedGameId, subscribeToGame, navigate, clearMatch]);

  const handlePlayClick = (preset: string) => {
    if (user) {
      joinQueue(preset);
      return;
    }
    openModal(() => joinQueue(preset));
  };

  if (matchmakingStatus === "queued") {
    return (
      <div className="flex flex-col items-center gap-4 py-12">
        <p className="text-lg font-semibold">Searching for an opponent…</p>
        <button
          onClick={() => leaveQueue()}
          className="rounded-md border border-gray-300 px-4 py-2 text-sm"
        >
          Cancel
        </button>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4 max-w-sm">
      {PRESETS.map((p) => (
        <button
          key={p.key}
          onClick={() => handlePlayClick(p.key)}
          className="rounded-md bg-blue-600 px-4 py-3 text-white font-medium"
        >
          {p.label}
        </button>
      ))}
    </div>
  );
}
