// The time-control presets the server accepts (api/internal/matchmaking/
// preset.go), grouped for display. `key` is what goes on the wire; `ms` is
// the group's initial clock, used for the placeholder clocks before a game.
// Shared by matchmaking (play-online) and friend challenges.
export const PRESET_GROUPS = [
  {
    category: "Bullet",
    options: [
      { label: "1 min", key: "1+0" },
      { label: "1 + 1", key: "1+1" },
      { label: "2 + 1", key: "2+1" },
    ],
    ms: 60000,
  },
  {
    category: "Blitz",
    options: [
      { label: "3 min", key: "3+0" },
      { label: "3 + 2", key: "3+2" },
      { label: "5 min", key: "5+0" },
    ],
    ms: 180000,
  },
  {
    category: "Rapid",
    options: [
      { label: "10 min", key: "10+0" },
      { label: "10 + 5", key: "10+5" },
      { label: "15 + 10", key: "15+10" },
    ],
    ms: 600000,
  },
];

export const ALL_PRESETS = PRESET_GROUPS.flatMap((g) => g.options);

/** A preset's starting clock: "3+2" is 180000. */
export function presetInitialMs(key: string): number {
  return Number(key.split("+")[0]) * 60_000;
}

/** The preset a game was played at, or undefined if it isn't one. */
export function presetForTimeControl(initialMs: number, incrementMs: number): string | undefined {
  const key = `${initialMs / 60_000}+${incrementMs / 1000}`;
  return ALL_PRESETS.some((p) => p.key === key) ? key : undefined;
}

/** "10 min (Rapid)" for a preset key. */
export function describePreset(key: string): string {
  for (const group of PRESET_GROUPS) {
    const option = group.options.find((o) => o.key === key);
    if (option) return `${option.label} (${group.category})`;
  }
  return key;
}
