import { formatClock } from "./format-clock";

interface ClockDisplayProps {
  isWhite: boolean;
  ms: number;
  isActive: boolean;
}

export function ClockDisplay({ isWhite, ms, isActive }: ClockDisplayProps) {
  const isLow = ms < 30_000;

  return (
    <div
      className={[
        "flex h-11 w-32 items-center justify-end rounded-sm px-3 font-mono text-2xl tabular-nums",
        isWhite && !isActive ? "bg-clock-bg" : "bg-nav-bg",
        // One text colour, not two. This used to set text-clock-white-d /
        // text-clock-black-d and then text-clock-active / -disabled on top,
        // but the latter pair had no tokens behind them so they did nothing.
        isActive && isLow
          ? "text-red-400"
          : isActive
            ? "text-clock-active"
            : "text-clock-disabled",
      ].join(" ")}
    >
      {formatClock(ms)}
    </div>
  );
}
