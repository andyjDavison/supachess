import { Link } from "react-router";

const buttonClass =
  "text-xs px-3 py-1.5 rounded-md text-text bg-linear-to-b from-gradient-start to-gradient-end hover:from-hover-gradient-start hover:to-hover-gradient-end hover:cursor-pointer";

/**
 * Opens the conversation with this player - or a draft of one, if they've
 * never talked. /messages resolves which.
 */
export function MessageButton({ userId }: { userId: string }) {
  return (
    <Link to={`/messages?to=${encodeURIComponent(userId)}`} className={buttonClass}>
      Message
    </Link>
  );
}
