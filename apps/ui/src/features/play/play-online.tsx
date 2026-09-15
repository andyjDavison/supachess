import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../../stores/AuthContext";
import { useAuthModal } from "../../stores/AuthModalContext";
import { useGameStore } from "../game/store";
import { Chessboard, type ChessboardOptions } from "react-chessboard";
import black400 from "../../assets/black_400.png";
import white400 from "../../assets/white_400.png";
import { ClockDisplay } from "../game/clock-display";

type Tabs = "New Game" | "Games" | "Players";

type Preset = {
  label: string;
  key: string;
  category: string;
  ms: number;
};

const PRESET_GROUPS = [
  {
    category: "Bullet",
    options: [
      { label: "1 min", key: "1+0" },
      { label: "1 + 1", key: "1+1" },
      { label: "2 + 1", key: "2+1" },
    ],
    ms: 60000,
  },
  {
    category: "Blitz",
    options: [
      { label: "3 min", key: "3+0" },
      { label: "3 + 2", key: "3+2" },
      { label: "5 min", key: "5+0" },
    ],
    ms: 180000,
  },
  {
    category: "Rapid",
    options: [
      { label: "10 min", key: "10+0" },
      { label: "10 + 5", key: "10+5" },
      { label: "15 + 10", key: "15+10" },
    ],
    ms: 600000,
  },
];

const chessboardOptions: ChessboardOptions = {
  allowDragging: false,
  showAnimations: false,
  boardStyle: {
    borderRadius: 4,
    pointerEvents: "none",
    maxWidth: "100%",
  },
  darkSquareStyle: {
    background: "oklch(74% 0.238 322.16)",
  },
};

export default function PlayOnline() {
  const { user } = useAuth();
  const { openModal } = useAuthModal();
  const navigate = useNavigate();

  const [selectedTab, setSelectedTab] = useState<Tabs>("New Game");
  const [isOpen, setIsOpen] = useState<boolean>(false);
  const [selectedPreset, setSelectedPreset] = useState<Preset>({
    label: "10 min",
    key: "10+0",
    category: "Rapid",
    ms: 600000,
  });

  const matchmakingStatus = useGameStore((s) => s.matchmakingStatus);
  const matchedGameId = useGameStore((s) => s.matchedGameId);
  const opponent = useGameStore((s) => s.opponent);
  const joinQueue = useGameStore((s) => s.joinQueue);
  const leaveQueue = useGameStore((s) => s.leaveQueue);
  const subscribeToGame = useGameStore((s) => s.subscribeToGame);
  const clearMatch = useGameStore((s) => s.clearMatch);

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

  function handlePresetClick(
    { key, label }: { key: string; label: string },
    category: string,
    ms: number,
  ) {
    setSelectedPreset({ key, label, category, ms });
    setIsOpen(false);
  }

  return (
    <div className="flex items-center justify-center w-full h-full gap-7">
      <div className="flex flex-col max-h-full max-w-158 hover:cursor-default gap-2">
        <div className="flex gap-2 w-full h-9">
          <img src={black400} className="rounded-xs" />
          <p className="text-white text-xs font-extrabold">Opponent</p>
          <div className="flex-1 flex justify-end">
            <ClockDisplay
              isWhite={false}
              ms={selectedPreset.ms}
              isActive={false}
            />
          </div>
        </div>
        <Chessboard options={chessboardOptions} />
        <div className="flex gap-2 w-full h-9">
          <img src={white400} className="rounded-xs" />
          <p className="text-white text-xs font-extrabold">Player</p>
          <div className="flex-1 flex w-full justify-end">
            <ClockDisplay
              isWhite={true}
              ms={selectedPreset.ms}
              isActive={false}
            />
          </div>
        </div>
      </div>
      <div className="flex flex-col h-full flex-1 min-w-75 max-w-120">
        <div className="flex flex-row justify-center items-center bg-options-header rounded-t-sm h-17">
          <div
            onClick={() => setSelectedTab("New Game")}
            className={`flex items-center justify-center w-1/3 h-full ${selectedTab === "New Game" ? "bg-nav-bg text-text" : ""} hover:cursor-pointer hover:text-text rounded-tl-md`}
          >
            New Game
          </div>
          <div
            onClick={() => setSelectedTab("Games")}
            className={`flex items-center justify-center w-1/3 h-full ${selectedTab === "Games" ? "bg-nav-bg text-text" : ""} hover:cursor-pointer hover:text-text`}
          >
            Games
          </div>
          <div
            onClick={() => setSelectedTab("Players")}
            className={`flex items-center justify-center w-1/3 h-full ${selectedTab === "Players" ? "bg-nav-bg text-text" : ""} hover:cursor-pointer hover:text-text rounded-tr-md`}
          >
            Players
          </div>
        </div>
        {selectedTab === "New Game" && (
          <div className="flex flex-col flex-1 gap-2 p-4 bg-nav-bg rounded-b-sm overflow-y-auto">
            <button
              onClick={() => setIsOpen(!isOpen)}
              className="flex flex-row justify-center items-center gap-4 p-4 rounded-lg gap-.5 bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end hover:cursor-pointer text-text"
            >
              {`${selectedPreset.label} (${selectedPreset.category})`}
            </button>
            {isOpen && (
              <div className="flex flex-col gap-4">
                {PRESET_GROUPS.map((group) => (
                  <div
                    key={group.category}
                    className="flex flex-col items-start gap-1"
                  >
                    <p className="text-text">{group.category}</p>
                    <div className="flex w-full gap-2">
                      {group.options.map((option) => (
                        <button
                          key={option.key}
                          onClick={() =>
                            handlePresetClick(option, group.category, group.ms)
                          }
                          className={`text-sm w-1/3 justify-center items-center py-3 px-1 rounded-md bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end hover:cursor-pointer text-text
                        ${selectedPreset.key === option.key ? "border border-fuchsia-300" : ""}`}
                        >
                          {option.label}
                        </button>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            )}
            <button
              onClick={() => handlePlayClick(selectedPreset.key)}
              className="flex justify-center items-center gap-4 p-4 rounded-lg bg-linear-to-b from-fuchsia-400 to-fuchsia-500 hover:bg-linear-to-b hover:from-fuchsia-300 hover:to-fuchsia-500 hover:cursor-pointer text-text"
            >
              Start Game
            </button>
          </div>
        )}
        {selectedTab === "Games" && (
          <div className="flex flex-col flex-1 gap-2 p-4 bg-nav-bg rounded-b-sm overflow-y-auto"></div>
        )}
        {selectedTab === "Players" && (
          <div className="flex flex-col flex-1 gap-2 p-4 bg-nav-bg rounded-b-sm overflow-y-auto"></div>
        )}
      </div>
    </div>
  );
}
