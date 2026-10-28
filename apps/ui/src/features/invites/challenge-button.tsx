import { useState } from "react";
import { PRESET_GROUPS } from "../../lib/presets";
import { useInviteStore } from "./store";

const buttonClass =
  "text-xs px-3 py-1.5 rounded-md text-text bg-linear-to-b from-gradient-start to-gradient-end hover:from-hover-gradient-start hover:to-hover-gradient-end hover:cursor-pointer disabled:opacity-50 disabled:cursor-default";

/**
 * "Challenge" plus a preset picker. Colors are the server's coin flip, so
 * the time control is the only choice. Disabled while this user already
 * has a challenge out - the server allows one at a time.
 */
export function ChallengeButton({ userId, username }: { userId: string; username: string }) {
  const [open, setOpen] = useState(false);
  const busy = useInviteStore((s) => s.outgoing !== null || s.pendingOffer !== null);
  const offer = useInviteStore((s) => s.offer);

  return (
    <div className="relative">
      <button
        className={buttonClass}
        disabled={busy}
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
      >
        Challenge
      </button>
      {open && !busy && (
        <div
          role="menu"
          aria-label={`Challenge ${username}`}
          className="absolute right-0 top-full z-20 mt-1 flex w-56 flex-col gap-2 rounded-md bg-options-header p-3 shadow-lg"
        >
          {PRESET_GROUPS.map((group) => (
            <div key={group.category} className="flex flex-col gap-1">
              <p className="text-xs text-clock-disabled">{group.category}</p>
              <div className="flex gap-1">
                {group.options.map((preset) => (
                  <button
                    key={preset.key}
                    role="menuitem"
                    className={`${buttonClass} flex-1 px-1`}
                    onClick={() => {
                      offer(userId, username, preset.key);
                      setOpen(false);
                    }}
                  >
                    {preset.label}
                  </button>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
