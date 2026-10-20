import { useEffect, useRef } from "react";
import { useGameStore } from "./store";
import { pairMoves } from "./move-rows";

export function MoveList() {
  const moves = useGameStore((s) => s.moves);
  const endRef = useRef<HTMLLIElement>(null);

  // Keep the latest move in view as the game goes on. Tied to length
  // rather than the array itself so a resync that replaces equal-length
  // content doesn't yank the scroll position.
  //
  // scrollIntoView is called optionally because it isn't guaranteed to
  // exist - jsdom doesn't implement it. The scroll is a nicety, so a
  // missing implementation should no-op rather than throw mid-render.
  useEffect(() => {
    endRef.current?.scrollIntoView?.({ block: "end" });
  }, [moves.length]);

  if (moves.length === 0) {
    return <p className="text-sm text-clock-black-d">No moves yet</p>;
  }

  const rows = pairMoves(moves);

  return (
    <ol className="flex flex-col font-mono text-sm tabular-nums text-text">
      {rows.map((row) => (
        <li
          key={row.number}
          className="flex gap-2 px-2 py-1 odd:bg-options-header rounded-xs"
        >
          <span className="w-8 shrink-0 text-right text-clock-black-d">
            {row.number}.
          </span>
          <span className="flex-1 text-left">{row.white?.san ?? ""}</span>
          <span className="flex-1 text-left">{row.black?.san ?? ""}</span>
        </li>
      ))}
      {/* Scroll anchor. An <li> rather than a <div> - a div is not a
          valid child of <ol>. */}
      <li ref={endRef} aria-hidden="true" className="h-0" />
    </ol>
  );
}
