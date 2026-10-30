import { create } from "zustand";
import { boardTheme, type BoardThemeId } from "../lib/board-themes";

const STORAGE_KEY = "supachess.board-settings";

type Saved = { themeId: BoardThemeId; showCoordinates: boolean };
const DEFAULTS: Saved = { themeId: "supachess", showCoordinates: true };

// Storage can be missing or throw (private windows, blocked site data), so
// every access is guarded and a failure just means the defaults.
function load(): Saved {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return DEFAULTS;
    const saved = JSON.parse(raw) as Partial<Saved>;
    return {
      themeId: boardTheme(saved.themeId ?? DEFAULTS.themeId).id,
      showCoordinates: typeof saved.showCoordinates === "boolean" ? saved.showCoordinates : DEFAULTS.showCoordinates,
    };
  } catch {
    return DEFAULTS;
  }
}

function save(settings: Saved) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
  } catch {
    // Not persisted; still applies for this page load.
  }
}

interface BoardSettings extends Saved {
  setTheme: (id: BoardThemeId) => void;
  setShowCoordinates: (show: boolean) => void;
  /** Re-reads storage. For tests; the store loads once at startup. */
  reload: () => void;
}

/**
 * How the board looks: square colors and coordinates. Per browser until
 * there's a Settings page to store it on the account.
 */
export const useBoardSettings = create<BoardSettings>((set, get) => {
  const persist = (patch: Partial<Saved>) => {
    set(patch);
    const { themeId, showCoordinates } = get();
    save({ themeId, showCoordinates });
  };
  return {
    ...load(),
    setTheme: (themeId) => persist({ themeId }),
    setShowCoordinates: (showCoordinates) => persist({ showCoordinates }),
    reload: () => set(load()),
  };
});

/** The Chessboard options that come from the board settings. */
export function useBoardAppearance() {
  const themeId = useBoardSettings((s) => s.themeId);
  const showNotation = useBoardSettings((s) => s.showCoordinates);
  const theme = boardTheme(themeId);
  return {
    lightSquareStyle: { backgroundColor: theme.light },
    darkSquareStyle: { background: theme.dark },
    showNotation,
  };
}
