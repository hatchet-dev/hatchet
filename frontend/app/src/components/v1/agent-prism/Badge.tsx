import type { ComponentPropsWithRef, ReactElement, ReactNode } from "react";

import cn from "classnames";

import type { ComponentSize } from "./shared";

type BadgeSize = Extract<ComponentSize, "4" | "5" | "6" | "7">;

const sizeClasses: Record<BadgeSize, string> = {
  "4": "px-1 gap-1 h-4",
  "5": "px-1.5 gap-1 h-5",
  "6": "px-2 gap-1.5 h-6",
  "7": "px-2.5 gap-2 h-7",
};

const textSizes: Record<BadgeSize, string> = {
  "4": "text-xs leading-3",
  "5": "text-xs",
  "6": "text-sm",
  "7": "text-sm",
};

export type BadgeProps = ComponentPropsWithRef<"span"> & {
  /**
   * The content of the badge
   */
  label: ReactNode;

  /**
   * The size of the badge
   * @default "md"
   */
  size?: BadgeSize;

  /**
   * Optional icon to display at the start of the badge
   */
  iconStart?: ReactElement;

  /**
   * Optional icon to display at the end of the badge
   */
  iconEnd?: ReactElement;

  /**
   * Optional className for additional styling
   */
  className?: string;

  /**
   * Whether to render the badge without any default styles
   * @default false
   */
  unstyled?: boolean;
};

/**
 * An unstyled badge component that displays a label with an optional icon
 */
export const Badge = ({
  label,
  size = "4",
  iconStart,
  iconEnd,
  className = "",
  unstyled = false,
  ...rest
}: BadgeProps): ReactElement => {
  return (
    <span
      className={cn(
        "inline-flex min-w-0 items-center overflow-hidden rounded-md font-medium",
        sizeClasses[size],
        className,
        unstyled
          ? ""
          : "bg-agentprism-badge-default text-agentprism-badge-default-foreground",
      )}
      {...rest}
    >
      {iconStart && <span className="shrink-0">{iconStart}</span>}

      <span
        className={cn(
          textSizes[size],
          "min-w-0 max-w-full flex-shrink-0 truncate font-medium tracking-normal",
        )}
      >
        {label}
      </span>

      {iconEnd && <span className="shrink-0">{iconEnd}</span>}
    </span>
  );
};
