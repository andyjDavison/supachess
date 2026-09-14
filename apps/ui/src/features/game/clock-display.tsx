interface ClockDisplayProps {
  ms: number;
  isActive: boolean;
}

function formatClock(ms: number): string {
  const totalSeconds = Math.max(0, Math.ceil(ms / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
}

export function ClockDisplay({ ms, isActive }: ClockDisplayProps) {
  const isLow = ms < 30_000;

  return (
    <div
      className={[
        "rounded-md px-3 py-1.5 font-mono text-lg tabular-nums",
        isActive ? "bg-gray-900 text-white" : "bg-gray-100 text-gray-500",
        isActive && isLow ? "text-red-400" : "",
      ].join(" ")}
    >
      {formatClock(ms)}
    </div>
  );
}
