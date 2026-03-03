import type { TraceSpanCategory } from "@evilmartians/agent-prism-types";

import cn from "classnames";

import { Badge, type BadgeProps } from "./Badge";
import { getSpanCategoryIcon, getSpanCategoryLabel } from "./shared";

export interface SpanBadgeProps
  extends Omit<BadgeProps, "label" | "iconStart" | "iconEnd"> {
  category: TraceSpanCategory;
}

const badgeClasses: Record<TraceSpanCategory, string> = {
  llm_call: "bg-agentprism-badge-llm text-agentprism-badge-llm-foreground",
  tool_execution:
    "bg-agentprism-badge-tool text-agentprism-badge-tool-foreground",
  chain_operation:
    "bg-agentprism-badge-chain text-agentprism-badge-chain-foreground",
  retrieval:
    "bg-agentprism-badge-retrieval text-agentprism-badge-retrieval-foreground",
  embedding:
    "bg-agentprism-badge-embedding text-agentprism-badge-embedding-foreground",
  guardrail:
    "bg-agentprism-badge-guardrail text-agentprism-badge-guardrail-foreground",
  agent_invocation:
    "bg-agentprism-badge-agent text-agentprism-badge-agent-foreground",
  create_agent:
    "bg-agentprism-badge-create-agent text-agentprism-badge-create-agent-foreground",
  span: "bg-agentprism-badge-span text-agentprism-badge-span-foreground",
  event: "bg-agentprism-badge-event text-agentprism-badge-event-foreground",
  unknown:
    "bg-agentprism-badge-unknown text-agentprism-badge-unknown-foreground",
};

export const SpanBadge = ({
  category,
  className,
  ...props
}: SpanBadgeProps) => {
  const Icon = getSpanCategoryIcon(category);
  const label = getSpanCategoryLabel(category);

  return (
    <Badge
      className={cn(badgeClasses[category], className)}
      iconStart={<Icon className="size-2.5" />}
      {...props}
      label={label}
      unstyled
    />
  );
};
