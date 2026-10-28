/**
 * A short timestamp for a message: the time if it was today, otherwise the
 * date. `now` is a parameter so tests don't depend on the wall clock.
 */
export function formatMessageTime(sentAt: number, now: number = Date.now()): string {
  const sent = new Date(sentAt);
  const today = new Date(now);
  const sameDay =
    sent.getFullYear() === today.getFullYear() &&
    sent.getMonth() === today.getMonth() &&
    sent.getDate() === today.getDate();
  return sameDay
    ? sent.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" })
    : sent.toLocaleDateString([], { month: "short", day: "numeric" });
}
