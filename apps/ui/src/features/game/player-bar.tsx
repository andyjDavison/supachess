import type { ReactNode } from "react";
import { defaultPieces } from "react-chessboard";
import type { PieceType } from "../../lib/chess/material";
import { ClockDisplay } from "./clock-display";
import white400 from "../../assets/white_400.png";
import black400 from "../../assets/black_400.png";

type Color = "white" | "black";

/**
 * The row above or below the board: avatar, name and rating, the pieces
 * this player has captured with their material lead, and their clock.
 */
export function PlayerBar({
  color,
  name,
  rating,
  captured = [],
  advantage = 0,
  clock,
  pending = false,
}: {
  color: Color;
  name: ReactNode;
  rating?: number;
  /** The opponent's pieces this player has taken. */
  captured?: PieceType[];
  /** How far ahead in material this player is; shown only when > 0. */
  advantage?: number;
  clock?: { ms: number; isActive: boolean };
  /** No player here yet (e.g. while searching): a pulsing avatar. */
  pending?: boolean;
}) {
  return (
    <div className="flex h-11 w-full items-center gap-2" data-testid={`player-bar-${color}`}>
      <img
        src={color === "white" ? white400 : black400}
        alt=""
        className={`size-10 rounded-sm ${pending ? "animate-pulse opacity-60" : ""}`}
      />
      <div className="flex min-w-0 flex-col items-start">
        <p className="truncate text-sm font-extrabold text-text">
          {name}
          {rating !== undefined && <span className="ml-1 font-normal text-clock-disabled">({rating})</span>}
        </p>
        <CapturedPieces pieces={captured} color={color === "white" ? "b" : "w"} advantage={advantage} />
      </div>
      {clock && (
        <div className="ml-auto">
          <ClockDisplay isWhite={color === "white"} ms={clock.ms} isActive={clock.isActive} />
        </div>
      )}
    </div>
  );
}

/** Small overlapping pieces, grouped by type, then "+N". */
function CapturedPieces({ pieces, color, advantage }: { pieces: PieceType[]; color: "w" | "b"; advantage: number }) {
  if (pieces.length === 0 && advantage <= 0) return <div className="h-5" />;
  return (
    <div className="flex h-5 items-center" aria-label={`Captured: ${pieces.join(" ") || "none"}`}>
      {pieces.map((p, i) => {
        const render = defaultPieces[`${color}${p.toUpperCase()}`];
        const newGroup = i > 0 && pieces[i - 1] !== p;
        return (
          <span
            key={i}
            className={`size-5 shrink-0 ${i === 0 ? "" : newGroup ? "ml-1" : "-ml-2.5"} ${
              // Black pieces vanish on the dark chrome without an outline.
              color === "b" ? "drop-shadow-[0_0_1px_rgba(255,255,255,0.7)]" : ""
            }`}
          >
            {render?.({ svgStyle: { width: "100%", height: "100%" } })}
          </span>
        );
      })}
      {advantage > 0 && <span className="ml-1 text-xs text-clock-disabled">+{advantage}</span>}
    </div>
  );
}
