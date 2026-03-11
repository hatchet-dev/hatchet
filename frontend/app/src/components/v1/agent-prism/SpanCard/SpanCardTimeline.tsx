import { getTimelineData } from '../agent-prism-data';
import type { OtelSpanTree } from '../span-tree-type';
import { OtelStatusCode } from '@/lib/api/generated/data-contracts';
import cn from 'classnames';

interface SpanCardTimelineProps {
  spanCard: OtelSpanTree;
  minStart: number;
  maxEnd: number;
  className?: string;
}

const timelineBgColors: Record<OtelStatusCode, string> = {
  [OtelStatusCode.OK]: 'bg-success',
  [OtelStatusCode.UNSET]: 'bg-success',
  [OtelStatusCode.ERROR]: 'bg-danger',
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
        'bg-agentprism-secondary relative flex h-4 min-w-20 flex-1 rounded-md',
        className,
      )}
    >
      <span className="pointer-events-none absolute inset-x-1 top-1/2 h-1.5 -translate-y-1/2">
        <span
          className={`absolute h-full rounded-sm ${timelineBgColors[spanCard.status_code]}`}
          style={{
            left: `${startPercent}%`,
            width: `${widthPercent}%`,
          }}
        />
      </span>
    </span>
  );
};
