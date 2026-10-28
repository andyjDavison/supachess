import { useState } from "react";
import type { Game } from "../../../lib/ws/schemas";
import { MoveList } from "../../game/move-list";
import { GameChat } from "../../messages/game-chat";

const SUB_TABS = ["Moves", "Chat"] as const;
type SubTab = (typeof SUB_TABS)[number];

/** The game page's Analysis tab: the move list, with game chat beside it. */
export default function AnalysisSidePanel({ game }: { game: Game }) {
  const [tab, setTab] = useState<SubTab>("Moves");

  return (
    <div className="flex flex-col flex-1 min-h-0 bg-nav-bg rounded-b-sm">
      <div role="tablist" aria-label="Analysis" className="flex gap-4 border-b border-options-header px-4">
        {SUB_TABS.map((t) => (
          <button
            key={t}
            role="tab"
            aria-selected={tab === t}
            onClick={() => setTab(t)}
            className={`py-2 text-sm hover:cursor-pointer hover:text-text ${
              tab === t ? "border-b-2 border-accent text-text" : "text-clock-disabled"
            }`}
          >
            {t}
          </button>
        ))}
      </div>
      {tab === "Moves" ? (
        <div className="flex flex-col flex-1 gap-2 p-4 overflow-y-auto">
          <MoveList />
        </div>
      ) : (
        <GameChat game={game} />
      )}
    </div>
  );
}
