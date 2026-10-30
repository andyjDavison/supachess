import type { ReactNode } from "react";

export type PanelTab<T extends string> = { id: T; label: string; icon?: ReactNode };

/**
 * The header of a side panel: a row of tabs, the selected one merging
 * into the panel body below it.
 */
export function PanelTabs<T extends string>({
  tabs,
  selected,
  onSelect,
  label,
}: {
  tabs: readonly PanelTab<T>[];
  selected: T;
  onSelect: (id: T) => void;
  label: string;
}) {
  return (
    <div role="tablist" aria-label={label} className="flex h-16 shrink-0 rounded-t-md bg-options-header">
      {tabs.map((tab, i) => (
        <button
          key={tab.id}
          type="button"
          role="tab"
          aria-selected={selected === tab.id}
          onClick={() => onSelect(tab.id)}
          className={[
            "flex flex-1 flex-col items-center justify-center gap-1 text-xs hover:cursor-pointer hover:text-text",
            selected === tab.id ? "bg-nav-bg text-text" : "text-clock-disabled",
            i === 0 ? "rounded-tl-md" : "",
            i === tabs.length - 1 ? "rounded-tr-md" : "",
          ].join(" ")}
        >
          {tab.icon}
          {tab.label}
        </button>
      ))}
    </div>
  );
}
