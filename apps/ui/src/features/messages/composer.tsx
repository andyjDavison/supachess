import { useState, type KeyboardEvent } from "react";

/** The server's limit (domain.MaxMessageRunes), counted in characters. */
const MAX_MESSAGE_CHARS = 500;

/**
 * A message box shared by DMs and game chat. Enter sends, Shift+Enter is a
 * newline. Over-long or blank messages can't be sent; the server enforces
 * the same rule, this just saves the round trip.
 */
export function Composer({
  onSend,
  disabled = false,
  placeholder = "Write a message…",
}: {
  onSend: (body: string) => void;
  disabled?: boolean;
  placeholder?: string;
}) {
  const [body, setBody] = useState("");
  // Array.from counts code points, matching the server's rune count, so an
  // emoji counts as one character rather than two UTF-16 units.
  const length = Array.from(body).length;
  const canSend = !disabled && body.trim() !== "" && length <= MAX_MESSAGE_CHARS;

  const submit = () => {
    if (!canSend) return;
    onSend(body.trim());
    setBody("");
  };

  const onKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      submit();
    }
  };

  return (
    <div className="flex flex-col gap-1 border-t border-options-header p-2">
      <div className="flex items-end gap-2">
        <textarea
          aria-label="Message"
          value={body}
          onChange={(e) => setBody(e.target.value)}
          onKeyDown={onKeyDown}
          placeholder={placeholder}
          disabled={disabled}
          rows={2}
          className="flex-1 resize-none rounded-md border border-input-border bg-input-bg px-3 py-2 text-sm text-text placeholder:text-clock-disabled focus:border-accent focus:outline-none disabled:opacity-50"
        />
        <button
          onClick={submit}
          disabled={!canSend}
          className="rounded-md bg-accent-strong px-3 py-2 text-xs text-white hover:cursor-pointer hover:bg-accent disabled:cursor-default disabled:opacity-50"
        >
          Send
        </button>
      </div>
      {length > MAX_MESSAGE_CHARS * 0.8 && (
        <p className={`text-right text-[10px] ${length > MAX_MESSAGE_CHARS ? "text-red-300" : "text-clock-disabled"}`}>
          {length}/{MAX_MESSAGE_CHARS}
        </p>
      )}
    </div>
  );
}
