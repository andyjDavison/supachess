/** Seconds left until `expiresAt`, as m:ss, never negative. */
export function formatCountdown(expiresAt: number, now: number): string {
  const secs = Math.max(0, Math.ceil((expiresAt - now) / 1000));
  return `${Math.floor(secs / 60)}:${String(secs % 60).padStart(2, "0")}`;
}
