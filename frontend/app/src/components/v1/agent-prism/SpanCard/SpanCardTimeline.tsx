import type {
  TraceSpan,
  TraceSpanStatus,
} from '@/pages/main/v1/workflow-runs-v1/$run/v2components/step-run-detail/observability/agent-prism-types';
import { getTimelineData } from '@evilmartians/agent-prism-data';
import cn from 'classnames';

interface SpanCardTimelineProps {
  spanCard: TraceSpan;
  minStart: number;
  maxEnd: number;
  className?: string;
}

const timelineBgColors: Record<TraceSpanStatus, string> = {
  success: 'bg-success',
  pending: 'bg-success',
  warning: 'bg-success',
  error: 'bg-danger',
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
          className={`absolute h-full rounded-sm ${timelineBgColors[spanCard.status]}`}
          style={{
            left: `${startPercent}%`,
            width: `${widthPercent}%`,
          }}
        />
      </span>
    </span>
  );
};
