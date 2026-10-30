import type { ReactNode } from "react";
import { Link } from "react-router";

/**
 * The standalone /login and /register pages: the wordmark over a centered
 * card, with an optional footer row under the card.
 */
export function AuthLayout({ children, footer }: { children: ReactNode; footer?: ReactNode }) {
  return (
    <div className="flex min-h-full w-full flex-col items-center overflow-y-auto bg-bg px-4 py-10">
      <Link to="/" className="mb-8 text-4xl font-extrabold tracking-tight text-text">
        Supa<span className="text-accent">chess</span>
      </Link>
      <div className="w-full max-w-md overflow-hidden rounded-xl bg-nav-bg shadow-2xl">{children}</div>
      {footer && <div className="mt-6 text-sm text-clock-disabled">{footer}</div>}
    </div>
  );
}

/** "──── OR ────", then the providers we don't support yet. */
export function ProviderButtons({ providers }: { providers: readonly string[] }) {
  return (
    <>
      <div className="flex items-center gap-3 text-xs text-clock-disabled">
        <span className="h-px flex-1 bg-input-border" />
        OR
        <span className="h-px flex-1 bg-input-border" />
      </div>
      {providers.map((provider) => (
        <button
          key={provider}
          type="button"
          disabled
          title="Coming soon"
          className="h-11 w-full rounded-lg border border-input-border text-sm font-extrabold text-clock-disabled disabled:cursor-not-allowed"
        >
          Continue with {provider}
        </button>
      ))}
    </>
  );
}
