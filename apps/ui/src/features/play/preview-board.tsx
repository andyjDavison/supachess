import type { ReactNode } from "react";
import { Chessboard } from "react-chessboard";
import { PlayerBar } from "../game/player-bar";
import { BoardSettingsButton } from "../game/board-settings";
import { useBoardAppearance } from "../../stores/board-settings";

/**
 * The board column of the pages before a game: the starting position, not
 * interactive, framed by player bars like the real game page so moving
 * into a game doesn't shift the layout.
 */
export function PreviewBoard({
  opponent,
  me,
  clockMs,
  searching = false,
}: {
  opponent: ReactNode;
  me: ReactNode;
  clockMs?: number;
  searching?: boolean;
}) {
  const appearance = useBoardAppearance();
  const clock = clockMs === undefined ? undefined : { ms: clockMs, isActive: false };
  return (
    <div className="flex flex-col items-center gap-2 w-full [--board-chrome:6.5rem]">
      <div className="board-square mx-auto flex flex-col gap-2 aspect-auto!">
        <PlayerBar color="black" name={opponent} clock={clock} pending={searching} />
        <div className="relative">
          <Chessboard
            options={{
              id: "preview",
              allowDragging: false,
              showAnimations: false,
              boardStyle: { borderRadius: 2, pointerEvents: "none" },
              ...appearance,
            }}
          />
          <div className="absolute top-0 -right-11">
            <BoardSettingsButton />
          </div>
        </div>
        <PlayerBar color="white" name={me} clock={clock} />
      </div>
    </div>
  );
}
