import type { ButtonHTMLAttributes } from "react";
import { buttonClass, type ButtonSize, type ButtonVariant } from "./button-styles";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
  size?: ButtonSize;
};

/** The site's button. For a link styled as one, use buttonClass on a Link. */
export function Button({ variant = "primary", size = "md", className = "", type = "button", ...rest }: ButtonProps) {
  return <button type={type} className={buttonClass(variant, size, className)} {...rest} />;
}
