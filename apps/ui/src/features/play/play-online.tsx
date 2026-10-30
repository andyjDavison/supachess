import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../../stores/AuthContext";
import { useAuthModal } from "../../stores/AuthModalContext";
import { useGameStore } from "../game/store";
import { PRESET_GROUPS, describePreset, presetInitialMs } from "../../lib/presets";
import { PreviewBoard } from "./preview-board";
import { PanelTabs } from "../../components/ui/panel-tabs";
import { Button } from "../../components/ui/button";
import { ListIcon, PlusIcon, UsersIcon } from "../../components/navbar/icons";
import { formatClock } from "../game/format-clock";

const TABS = [
  { id: "New Game", label: "New Game", icon: <PlusIcon /> },
  { id: "Games", label: "Games", icon: <ListIcon /> },
  { id: "Players", label: "Players", icon: <UsersIcon /> },
] as const;
type Tab = (typeof TABS)[number]["id"];

/**
 * Pick a time control and find an opponent. Laid out like the game page -
 * board column plus side panel - and the search happens inside the panel,
 * so the page doesn't change shape while you wait or when the game starts.
 */
export default function PlayOnline() {
  const { user } = useAuth();
  const { openModal } = useAuthModal();
  const navigate = useNavigate();

  const [selectedTab, setSelectedTab] = useState<Tab>("New Game");
  const [preset, setPreset] = useState("10+0");

  const matchmakingStatus = useGameStore((s) => s.matchmakingStatus);
  const queuedPreset = useGameStore((s) => s.queuedPreset);
  const matchedGameId = useGameStore((s) => s.matchedGameId);
  const joinQueue = useGameStore((s) => s.joinQueue);
  const leaveQueue = useGameStore((s) => s.leaveQueue);
  const subscribeToGame = useGameStore((s) => s.subscribeToGame);
  const clearMatch = useGameStore((s) => s.clearMatch);
  const searching = matchmakingStatus === "queued";
  const shownPreset = (searching && queuedPreset) || preset;

  useEffect(() => {
    if (!matchedGameId) return;

    subscribeToGame(matchedGameId);
    navigate(`/game/${matchedGameId}`);
    clearMatch();
  }, [matchedGameId, subscribeToGame, navigate, clearMatch]);

  const handlePlayClick = () => {
    if (user) {
      joinQueue(preset);
      return;
    }
    openModal(() => joinQueue(preset));
  };

  return (
    <div className="game-layout">
      <div className="board-column">
        <PreviewBoard
          opponent={searching ? "Searching…" : "Opponent"}
          me={user?.username ?? "Player"}
          clockMs={presetInitialMs(shownPreset)}
          searching={searching}
        />
      </div>
      <div className="side-panel">
        <PanelTabs tabs={TABS} selected={selectedTab} onSelect={setSelectedTab} label="Play online" />
        <div className="flex flex-1 flex-col gap-4 overflow-y-auto rounded-b-md bg-nav-bg p-4">
          {selectedTab === "New Game" &&
            (searching ? (
              <SearchingPanel preset={shownPreset} onCancel={leaveQueue} />
            ) : (
              <>
                <PresetPicker selected={preset} onSelect={setPreset} />
                <Button size="lg" onClick={handlePlayClick} className="mt-auto w-full">
                  Play
                </Button>
              </>
            ))}
          {selectedTab !== "New Game" && <p className="text-sm text-clock-disabled">Coming soon.</p>}
        </div>
      </div>
    </div>
  );
}

function PresetPicker({ selected, onSelect }: { selected: string; onSelect: (key: string) => void }) {
  return (
    <div className="flex flex-col gap-4">
      {PRESET_GROUPS.map((group) => (
        <div key={group.category} role="radiogroup" aria-label={group.category} className="flex flex-col items-start gap-2">
          <p className="text-sm font-extrabold text-text">{group.category}</p>
          <div className="grid w-full grid-cols-3 gap-2">
            {group.options.map((option) => (
              <button
                key={option.key}
                type="button"
                role="radio"
                aria-checked={selected === option.key}
                onClick={() => onSelect(option.key)}
                className={`h-11 rounded-md bg-options-button text-sm font-extrabold text-text hover:cursor-pointer hover:bg-hover-gradient-start ${
                  selected === option.key ? "ring-2 ring-accent" : ""
                }`}
              >
                {option.label}
              </button>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

function SearchingPanel({ preset, onCancel }: { preset: string; onCancel: () => void }) {
  const [seconds, setSeconds] = useState(0);
  useEffect(() => {
    const id = setInterval(() => setSeconds((s) => s + 1), 1000);
    return () => clearInterval(id);
  }, []);

  return (
    <div role="status" aria-label="Matchmaking" className="flex flex-1 flex-col items-center justify-center gap-4 text-center">
      <span aria-hidden="true" className="size-12 animate-spin rounded-full border-4 border-options-button border-t-accent" />
      <p className="text-lg font-extrabold">Searching for an opponent…</p>
      <p className="text-sm text-clock-disabled">{describePreset(preset)}</p>
      <p aria-label="Time searching" className="font-mono text-2xl tabular-nums">
        {formatClock(seconds * 1000)}
      </p>
      <Button variant="secondary" onClick={onCancel} className="w-full">
        Cancel
      </Button>
    </div>
  );
}
