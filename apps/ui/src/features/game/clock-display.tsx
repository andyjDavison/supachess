interface ClockDisplayProps {
  isWhite: boolean;
  ms: number;
  isActive: boolean;
}

function formatClock(ms: number): string {
  const totalSeconds = Math.max(0, Math.ceil(ms / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
}

export function ClockDisplay({ isWhite, ms, isActive }: ClockDisplayProps) {
  const isLow = ms < 30_000;

  return (
    <div
      className={[
        "flex items-center justify-end rounded-xs w-35 px-2 py-3 font-mono text-2xl tabular-nums",
        isWhite && !isActive
          ? "bg-clock-bg text-clock-white-d"
          : "bg-nav-bg text-clock-black-d",
        isActive ? "text-clock-active" : "text-clock-disabled",
        isActive && isLow ? "text-red-400" : "",
      ].join(" ")}
    >
      {formatClock(ms)}
    </div>
  );
}
