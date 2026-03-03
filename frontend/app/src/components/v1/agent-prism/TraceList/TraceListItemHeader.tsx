import type { TraceRecord } from "@evilmartians/agent-prism-types";

import type { AvatarProps } from "../Avatar";

import { Avatar } from "../Avatar";
import { Badge } from "../Badge";

interface TraceListItemHeaderProps {
  trace: TraceRecord;
  avatar?: AvatarProps;
}

export const TraceListItemHeader = ({
  trace,
  avatar,
}: TraceListItemHeaderProps) => {
  return (
    <header className="flex w-full min-w-0 flex-wrap items-center justify-between gap-2">
      <div className="flex min-w-0 items-center gap-1.5 overflow-hidden">
        {avatar && <Avatar size="4" {...avatar} />}

        <h3 className="text-agentprism-muted-foreground max-w-full truncate text-sm">
          {trace.name}
        </h3>
      </div>

      <div className="flex items-center gap-2">
        <Badge
          size="4"
          label={
            trace.spansCount === 1 ? "1 span" : `${trace.spansCount} spans`
          }
        />
      </div>
    </header>
  );
};
