import type { ComponentPropsWithRef, ReactElement } from "react";

import cn from "classnames";

import type { ComponentSize } from "./shared";

import { ROUNDED_CLASSES } from "./shared";

type ButtonSize = Extract<
  ComponentSize,
  "6" | "7" | "8" | "9" | "10" | "11" | "12" | "16"
>;

type ButtonVariant =
  | "brand"
  | "primary"
  | "outlined"
  | "secondary"
  | "ghost"
  | "destructive"
  | "success";

const BASE_CLASSES =
  "inline-flex items-center justify-center font-medium transition-all duration-200";

const sizeClasses = {
  "6": "h-6 px-2 gap-1 text-xs",
  "7": "h-7 px-2 gap-1 text-xs",
  "8": "h-8 px-2 gap-1 text-xs",
  "9": "h-9 px-2.5 gap-2 text-sm",
  "10": "h-10 px-4 gap-2 text-sm",
  "11": "h-11 px-5 gap-3 text-base",
  "12": "h-12 px-5 gap-2.5 text-base",
  "16": "h-16 px-7 gap-3 text-lg",
};

const variantClasses: Record<ButtonVariant, string> = {
  brand: "text-agentprism-brand-foreground bg-agentprism-brand",
  primary: "text-agentprism-primary-foreground bg-agentprism-primary",
  outlined:
    "border border bg-transparent text-agentprism-foreground border-agentprism-foreground",
  secondary: "bg-agentprism-secondary text-agentprism-secondary-foreground",
  ghost: "bg-transparent text-agentprism-foreground",
  destructive: "bg-agentprism-error text-agentprism-primary-foreground",
  success: "bg-agentprism-success text-agentprism-primary-foreground",
};

export type ButtonProps = ComponentPropsWithRef<"button"> & {
  /**
   * The size of the button
   * @default "6"
   */
  size?: ButtonSize;

  /**
   * The border radius of the button
   * @default "md"
   */
  rounded?: "none" | "sm" | "md" | "lg" | "full";

  /**
   * The visual variant of the button
   * @default "primary"
   */
  variant?: ButtonVariant;

  /**
   * Makes the button full width
   * @default false
   */
  fullWidth?: boolean;

  /**
   * Optional icon to display at the start of the button
   */
  iconStart?: ReactElement;

  /**
   * Optional icon to display at the end of the button
   */
  iconEnd?: ReactElement;
};

export const Button = ({
  children,
  size = "6",
  rounded = "md",
  variant = "primary",
  fullWidth = false,
  disabled = false,
  iconStart,
  iconEnd,
  type = "button",
  onClick,
  className = "",
  ...rest
}: ButtonProps) => {
  const widthClass = fullWidth ? "w-full" : "";
  const stateClasses = disabled
    ? "cursor-not-allowed opacity-50"
    : "hover:opacity-70";

  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled}
      className={cn(
        BASE_CLASSES,
        sizeClasses[size],
        ROUNDED_CLASSES[rounded],
        variantClasses[variant],
        widthClass,
        stateClasses,
        className,
      )}
      {...rest}
    >
      {iconStart && <span className="mr-1">{iconStart}</span>}
      {children}
      {iconEnd && <span className="ml-1">{iconEnd}</span>}
    </button>
  );
};
