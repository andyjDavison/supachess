// Class strings for Button, in a .ts file so links can share them
// (react-refresh wants .tsx files to export only components).

export type ButtonVariant = "primary" | "secondary" | "ghost";
export type ButtonSize = "sm" | "md" | "lg";

const BASE =
  "inline-flex items-center justify-center gap-2 rounded-lg font-extrabold transition-[transform,box-shadow] hover:cursor-pointer disabled:cursor-not-allowed disabled:opacity-50";

// The chunky look: a darker 4px bottom edge that a press pushes into.
const VARIANTS: Record<ButtonVariant, string> = {
  primary:
    "bg-linear-to-b from-accent to-accent-strong text-white shadow-[0_4px_0_0_var(--color-accent-shadow)] hover:from-accent-hover enabled:active:translate-y-0.5 enabled:active:shadow-[0_2px_0_0_var(--color-accent-shadow)]",
  secondary:
    "bg-linear-to-b from-gradient-start to-gradient-end text-text shadow-[0_4px_0_0_var(--color-button-shadow)] hover:from-hover-gradient-start hover:to-hover-gradient-end enabled:active:translate-y-0.5 enabled:active:shadow-[0_2px_0_0_var(--color-button-shadow)]",
  ghost: "text-clock-disabled hover:bg-options-button hover:text-text",
};

const SIZES: Record<ButtonSize, string> = {
  sm: "h-9 px-3 text-sm",
  md: "h-11 px-4 text-base",
  lg: "h-15 px-6 text-xl",
};

export function buttonClass(variant: ButtonVariant = "primary", size: ButtonSize = "md", extra = ""): string {
  return [BASE, VARIANTS[variant], SIZES[size], extra].filter(Boolean).join(" ");
}
