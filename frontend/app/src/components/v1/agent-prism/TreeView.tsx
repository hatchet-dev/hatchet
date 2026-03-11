import type { SpanCardViewOptions } from './SpanCard/SpanCard';
import { SpanCard } from './SpanCard/SpanCard';
import { flattenSpans, findTimeRange } from './agent-prism-data';
import type { AgentPrismTraceSpan } from './agent-prism-types';
import cn from 'classnames';
import { type FC } from 'react';

interface TreeViewProps {
  spans: AgentPrismTraceSpan[];
  className?: string;
  selectedSpan?: AgentPrismTraceSpan;
  onSpanSelect?: (span: AgentPrismTraceSpan) => void;
  expandedSpansIds: string[];
  onExpandSpansIdsChange: (ids: string[]) => void;
  spanCardViewOptions?: SpanCardViewOptions;
}

export const TreeView: FC<TreeViewProps> = ({
  spans,
  onSpanSelect,
  className = '',
  selectedSpan,
  expandedSpansIds,
  onExpandSpansIdsChange,
  spanCardViewOptions,
}) => {
  const allCards = flattenSpans(spans);
  const { minStart, maxEnd } = findTimeRange(allCards);

  return (
    <div className="w-full min-w-0 px-4">
      <ul
        className={cn(className, 'overflow-x-auto pt-2')}
        role="tree"
        aria-label="Hierarchical card list"
      >
        {spans.map((span, idx) => (
          <SpanCard
            key={span.id}
            data={span}
            level={0}
            selectedSpan={selectedSpan}
            onSpanSelect={onSpanSelect}
            minStart={minStart}
            maxEnd={maxEnd}
            isLastChild={idx === spans.length - 1}
            expandedSpansIds={expandedSpansIds}
            onExpandSpansIdsChange={onExpandSpansIdsChange}
            viewOptions={spanCardViewOptions}
          />
        ))}
      </ul>
    </div>
  );
};
