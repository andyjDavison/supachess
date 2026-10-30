import { forwardRef, useId, type InputHTMLAttributes, type ReactNode } from "react";

type TextFieldProps = InputHTMLAttributes<HTMLInputElement> & {
  label: string;
  error?: string;
  icon?: ReactNode;
};

/**
 * A labelled input with its validation message. The label is visually
 * hidden: the placeholder carries it on screen, the label for assistive
 * tech. Forwards its ref so react-hook-form's register() works.
 */
export const TextField = forwardRef<HTMLInputElement, TextFieldProps>(function TextField(
  { label, error, icon, id, className = "", ...rest },
  ref,
) {
  const generated = useId();
  const inputId = id ?? generated;
  const errorId = `${inputId}-error`;

  return (
    <div className="flex flex-col items-start gap-1 text-left">
      <label htmlFor={inputId} className="sr-only">
        {label}
      </label>
      <div className="relative w-full">
        {icon && <span className="pointer-events-none absolute inset-y-0 left-3 flex items-center text-clock-disabled">{icon}</span>}
        <input
          ref={ref}
          id={inputId}
          aria-invalid={!!error}
          aria-describedby={error ? errorId : undefined}
          className={`h-12 w-full rounded-md border bg-input-bg px-3 text-sm font-semibold text-text placeholder:text-clock-disabled focus:outline-none focus:ring-2 focus:ring-accent ${
            icon ? "pl-10" : ""
          } ${error ? "border-danger" : "border-input-border hover:border-clock-disabled"} ${className}`}
          {...rest}
        />
      </div>
      {error && (
        <p id={errorId} role="alert" className="text-sm text-danger">
          {error}
        </p>
      )}
    </div>
  );
});
