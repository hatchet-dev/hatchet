import { SpanBadge } from '../SpanBadge';
import type { TraceSpan } from '@evilmartians/agent-prism-types';

interface SpanCardBagdesProps {
  data: TraceSpan;
}

export const SpanCardBadges = ({ data }: SpanCardBagdesProps) => {
  return (
    <div className="flex flex-wrap items-center justify-start gap-1">
      <SpanBadge category={data.type} />
    </div>
  );
};
