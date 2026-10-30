// Board square colors the settings wheel offers. "supachess" is the
// original look: the board library's default light square with the accent
// as the dark one.

export const BOARD_THEMES = [
  { id: "supachess", name: "Supachess", light: "#f0d9b5", dark: "var(--color-accent)" },
  { id: "green", name: "Green", light: "#ebecd0", dark: "#739552" },
  { id: "brown", name: "Brown", light: "#f0d9b5", dark: "#b58863" },
  { id: "blue", name: "Blue", light: "#dee3e6", dark: "#8ca2ad" },
  { id: "gray", name: "Gray", light: "#e8e8e8", dark: "#9a9a9a" },
] as const;

export type BoardThemeId = (typeof BOARD_THEMES)[number]["id"];
export type BoardTheme = (typeof BOARD_THEMES)[number];

export function boardTheme(id: string): BoardTheme {
  return BOARD_THEMES.find((t) => t.id === id) ?? BOARD_THEMES[0];
}
