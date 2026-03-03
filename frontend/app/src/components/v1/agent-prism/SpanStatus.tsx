import type { TraceSpanStatus } from "@evilmartians/agent-prism-types";
import type { ComponentPropsWithRef } from "react";

import cn from "classnames";
import { Check, Ellipsis, Info, TriangleAlert } from "lucide-react";

type StatusVariant = "dot" | "badge";

export type StatusProps = ComponentPropsWithRef<"div"> & {
  status: TraceSpanStatus;
  variant?: StatusVariant;
};

const STATUS_COLORS_DOT: Record<TraceSpanStatus, string> = {
  success: "bg-agentprism-success",
  error: "bg-agentprism-error",
  pending: "bg-agentprism-pending",
  warning: "bg-agentprism-warning",
};

const STATUS_COLORS_BADGE: Record<TraceSpanStatus, string> = {
  success:
    "bg-agentprism-success-muted text-agentprism-success-muted-foreground",
  error: "bg-agentprism-error-muted text-agentprism-error-muted-foreground",
  pending:
    "bg-agentprism-pending-muted text-agentprism-pending-muted-foreground",
  warning:
    "bg-agentprism-warning-muted text-agentprism-warning-muted-foreground",
};

export const SpanStatus = ({
  status,
  variant = "dot",
  ...rest
}: StatusProps) => {
  const title = `Status: ${status}`;

  return (
    <div className="flex size-4 items-center justify-center" {...rest}>
      {variant === "dot" ? (
        <SpanStatusDot status={status} title={title} />
      ) : (
        <SpanStatusBadge status={status} title={title} />
      )}
    </div>
  );
};

interface StatusWithTitleProps extends StatusProps {
  title: string;
}

const SpanStatusDot = ({ status, title }: StatusWithTitleProps) => {
  return (
    <span
      className={cn("block size-1.5 rounded-full", STATUS_COLORS_DOT[status])}
      aria-label={title}
      title={title}
    />
  );
};

const SpanStatusBadge = ({ status, title }: StatusWithTitleProps) => {
  return (
    <span
      className={cn(
        "inline-flex items-center justify-center",
        "h-3.5 w-4 rounded",
        STATUS_COLORS_BADGE[status],
      )}
      aria-label={title}
      title={title}
    >
      {status === "success" && <Check className="size-2.5" aria-hidden />}
      {status === "error" && <TriangleAlert className="size-2.5" aria-hidden />}
      {status === "warning" && <Info className="size-2.5" aria-hidden />}
      {status === "pending" && <Ellipsis className="size-2.5" aria-hidden />}
    </span>
  );
};
