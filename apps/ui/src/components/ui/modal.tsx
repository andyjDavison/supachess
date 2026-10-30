import { useEffect, type ReactNode } from "react";

/**
 * A centered dialog over a dimmed page. Escape and a click on the backdrop
 * call onClose; leave onClose out for a dialog that must be answered.
 */
export function Modal({
  labelledBy,
  onClose,
  className = "",
  children,
}: {
  labelledBy: string;
  onClose?: () => void;
  className?: string;
  children: ReactNode;
}) {
  useEffect(() => {
    if (!onClose) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" onClick={onClose}>
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={labelledBy}
        onClick={(e) => e.stopPropagation()}
        className={`w-full max-w-sm overflow-hidden rounded-xl bg-nav-bg text-text shadow-2xl ${className}`}
      >
        {children}
      </div>
    </div>
  );
}
