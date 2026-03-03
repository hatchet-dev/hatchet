import type { KeyboardEvent, MouseEvent } from "react";

import * as Collapsible from "@radix-ui/react-collapsible";
import { ChevronDown, ChevronRight } from "lucide-react";

interface SpanCardToggleProps {
  isExpanded: boolean;
  title: string;
  onToggleClick: (e: MouseEvent | KeyboardEvent) => void;
}

export const SpanCardToggle = ({
  isExpanded,
  title,
  onToggleClick,
}: SpanCardToggleProps) => (
  <Collapsible.Trigger asChild>
    <button
      className="flex h-4 w-5 shrink-0 items-center justify-center"
      onClick={onToggleClick}
      onKeyDown={onToggleClick}
      aria-label={`${isExpanded ? "Collapse" : "Expand"} ${title} children`}
      aria-expanded={isExpanded}
      type="button"
    >
      {isExpanded ? (
        <ChevronDown
          aria-hidden="true"
          className="text-agentprism-muted-foreground size-3"
        />
      ) : (
        <ChevronRight
          aria-hidden="true"
          className="text-agentprism-muted-foreground size-3"
        />
      )}
    </button>
  </Collapsible.Trigger>
);
