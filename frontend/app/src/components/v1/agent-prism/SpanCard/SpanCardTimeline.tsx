import { getTimelineData } from '@/pages/main/v1/workflow-runs-v1/$run/v2components/step-run-detail/observability/agent-prism-data';
import type {
  TraceSpan,
  OpenTelemetryStatusCode,
} from '@/pages/main/v1/workflow-runs-v1/$run/v2components/step-run-detail/observability/agent-prism-types';
import cn from 'classnames';

interface SpanCardTimelineProps {
  spanCard: TraceSpan;
  minStart: number;
  maxEnd: number;
  className?: string;
}

const timelineBgColors: Record<OpenTelemetryStatusCode, string> = {
  STATUS_CODE_OK: 'bg-success',
  STATUS_CODE_UNSET: 'bg-success',
  STATUS_CODE_ERROR: 'bg-danger',
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
