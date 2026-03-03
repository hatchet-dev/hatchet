import type { TraceRecord } from "@evilmartians/agent-prism-types";
import type { KeyboardEvent } from "react";

import cn from "classnames";
import { useCallback } from "react";

import type { AvatarProps } from "../Avatar";
import type { BadgeProps } from "../Badge";

import { Badge } from "../Badge";
import { PriceBadge } from "../PriceBadge";
import { TimestampBadge } from "../TimestampBadge";
import { TokensBadge } from "../TokensBadge";
import { TraceListItemHeader } from "./TraceListItemHeader";

interface TraceListItemProps {
  trace: TraceRecord;
  badges?: Array<BadgeProps>;
  avatar?: AvatarProps;
  onClick?: () => void;
  isSelected?: boolean;
  showDescription?: boolean;
}

export const TraceListItem = ({
  trace,
  avatar,
  onClick,
  badges,
  isSelected,
  showDescription = true,
}: TraceListItemProps) => {
  const handleKeyDown = useCallback(
    (e: KeyboardEvent): void => {
      if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        onClick?.();
      }
    },
    [onClick],
  );

  const { name, agentDescription, totalCost, totalTokens, startTime } = trace;

  return (
    <div
      className={cn(
        "group w-full",
        "flex flex-col gap-2 p-4",
        "cursor-pointer",
        isSelected
          ? "bg-agentprism-secondary/75 dark:bg-agentprism-muted/80"
          : "bg-agentprism-background hover:bg-agentprism-secondary/45 dark:hover:bg-agentprism-muted/70",
      )}
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={handleKeyDown}
      aria-label={`Select trace ${name}`}
    >
      <TraceListItemHeader trace={trace} avatar={avatar} />

      <div className="flex flex-wrap items-center gap-2">
        {showDescription && (
          <span className="text-agentprism-muted-foreground mr-4 max-w-full truncate text-sm">
            {agentDescription}
          </span>
        )}

        {typeof totalCost === "number" && <PriceBadge cost={totalCost} />}

        {typeof totalTokens === "number" && (
          <TokensBadge tokensCount={totalTokens} />
        )}

        {badges?.map((badge, index) => (
          <Badge key={index} size="4" label={badge.label} />
        ))}

        {typeof startTime === "number" && (
          <TimestampBadge timestamp={startTime} />
        )}
      </div>
    </div>
  );
};
