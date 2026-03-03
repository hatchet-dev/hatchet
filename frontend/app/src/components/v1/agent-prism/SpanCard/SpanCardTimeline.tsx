import type {
  TraceSpan,
  TraceSpanCategory,
} from "@evilmartians/agent-prism-types";

import { getTimelineData } from "@evilmartians/agent-prism-data";
import cn from "classnames";

interface SpanCardTimelineProps {
  spanCard: TraceSpan;
  minStart: number;
  maxEnd: number;
  className?: string;
}

const timelineBgColors: Record<TraceSpanCategory, string> = {
  llm_call: "bg-agentprism-timeline-llm",
  agent_invocation: "bg-agentprism-timeline-agent",
  tool_execution: "bg-agentprism-timeline-tool",
  chain_operation: "bg-agentprism-timeline-chain",
  retrieval: "bg-agentprism-timeline-retrieval",
  embedding: "bg-agentprism-timeline-embedding",
  guardrail: "bg-agentprism-timeline-guardrail",
  create_agent: "bg-agentprism-timeline-create-agent",
  span: "bg-agentprism-timeline-span",
  event: "bg-agentprism-timeline-event",
  unknown: "bg-agentprism-timeline-unknown",
};

export const SpanCardTimeline = ({
  spanCard,
  minStart,
  maxEnd,
  className,
}: SpanCardTimelineProps) => {
  const { startPercent, widthPercent } = getTimelineData({
    spanCard,
    minStart,
    maxEnd,
  });

  return (
    <span
      className={cn(
        "bg-agentprism-secondary relative flex h-4 min-w-20 flex-1 rounded-md",
        className,
      )}
    >
      <span className="pointer-events-none absolute inset-x-1 top-1/2 h-1.5 -translate-y-1/2">
        <span
          className={`absolute h-full rounded-sm ${timelineBgColors[spanCard.type]}`}
          style={{
            left: `${startPercent}%`,
            width: `${widthPercent}%`,
          }}
        />
      </span>
    </span>
  );
};
