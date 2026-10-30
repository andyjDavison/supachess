import { useState } from "react";
import { Chessboard } from "react-chessboard";
import { Modal } from "../../components/ui/modal";
import { Button } from "../../components/ui/button";
import { GearIcon } from "../../components/navbar/icons";
import { BOARD_THEMES } from "../../lib/board-themes";
import { useBoardAppearance, useBoardSettings } from "../../stores/board-settings";

/** The gear beside the board, and the settings modal it opens. */
export function BoardSettingsButton() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button
        type="button"
        aria-label="Board settings"
        title="Board settings"
        onClick={() => setOpen(true)}
        className="flex size-9 items-center justify-center rounded-md text-clock-disabled hover:cursor-pointer hover:bg-options-button hover:text-text"
      >
        <GearIcon className="size-6" />
      </button>
      {open && <BoardSettingsModal onClose={() => setOpen(false)} />}
    </>
  );
}

export function BoardSettingsModal({ onClose }: { onClose: () => void }) {
  const { themeId, showCoordinates, setTheme, setShowCoordinates } = useBoardSettings();
  const appearance = useBoardAppearance();

  return (
    <Modal labelledBy="board-settings-title" onClose={onClose} className="max-w-md">
      <div className="flex items-center justify-between bg-options-header px-6 py-4">
        <h2 id="board-settings-title" className="text-xl font-extrabold">
          Board Settings
        </h2>
        <Button variant="ghost" size="sm" onClick={onClose} aria-label="Close">
          ✕
        </Button>
      </div>
      <div className="flex flex-col gap-5 p-6 text-left">
        <div className="mx-auto w-56">
          <Chessboard
            options={{ id: "settings-preview", allowDragging: false, showAnimations: false, boardStyle: { pointerEvents: "none" }, ...appearance }}
          />
        </div>

        <fieldset>
          <legend className="mb-2 text-sm font-extrabold">Board</legend>
          <div role="radiogroup" aria-label="Board theme" className="grid grid-cols-5 gap-2">
            {BOARD_THEMES.map((theme) => (
              <button
                key={theme.id}
                type="button"
                role="radio"
                aria-checked={themeId === theme.id}
                aria-label={theme.name}
                title={theme.name}
                onClick={() => setTheme(theme.id)}
                className={`grid aspect-square grid-cols-2 overflow-hidden rounded-md hover:cursor-pointer ${
                  themeId === theme.id ? "ring-2 ring-accent ring-offset-2 ring-offset-nav-bg" : ""
                }`}
              >
                <span style={{ background: theme.light }} />
                <span style={{ background: theme.dark }} />
                <span style={{ background: theme.dark }} />
                <span style={{ background: theme.light }} />
              </button>
            ))}
          </div>
        </fieldset>

        <label className="flex items-center justify-between text-sm font-extrabold hover:cursor-pointer">
          Show coordinates
          <input
            type="checkbox"
            checked={showCoordinates}
            onChange={(e) => setShowCoordinates(e.target.checked)}
            className="size-4 accent-[var(--color-accent)]"
          />
        </label>
      </div>
    </Modal>
  );
}
