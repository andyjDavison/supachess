/**
 * Formats a remaining-time value in milliseconds as m:ss.
 *
 * Rounds up, so a clock showing "1:00" has not yet reached zero - the
 * display only reads "0:00" when the time really is gone. Clamped at zero
 * so a clock that overran while a message was in flight never renders
 * negative.
 *
 * Lives apart from clock-display.tsx so it is importable without tripping
 * react-refresh's components-only rule for that file.
 */
export function formatClock(ms: number): string {
  const totalSeconds = Math.max(0, Math.ceil(ms / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
}
