import type { SpanCardViewOptions } from './SpanCard/SpanCard';
import { SpanCard } from './SpanCard/SpanCard';
import { findTimeRange } from './agent-prism-data';
import type { OtelSpanTree } from './span-tree-type';
import cn from 'classnames';
import { type FC } from 'react';

interface TreeViewProps {
  spanTree: OtelSpanTree;
  className?: string;
  selectedSpan?: OtelSpanTree;
  onSpanSelect?: (span: OtelSpanTree) => void;
  expandedSpansIds: string[];
  onExpandSpansIdsChange: (ids: string[]) => void;
  spanCardViewOptions?: SpanCardViewOptions;
}

export const TreeView: FC<TreeViewProps> = ({
  spanTree,
  onSpanSelect,
  className = '',
  selectedSpan,
  expandedSpansIds,
  onExpandSpansIdsChange,
  spanCardViewOptions,
}) => {
  const { minStart, maxEnd } = findTimeRange(spanTree);

  return (
    <div className="w-full min-w-0 px-4">
      <ul
        className={cn(className, 'overflow-x-auto pt-2')}
        role="tree"
        aria-label="Hierarchical card list"
      >
        <SpanCard
          key={spanTree.span_id}
          data={spanTree}
          level={0}
          selectedSpan={selectedSpan}
          onSpanSelect={onSpanSelect}
          minStart={minStart}
          maxEnd={maxEnd}
          isLastChild={true}
          expandedSpansIds={expandedSpansIds}
          onExpandSpansIdsChange={onExpandSpansIdsChange}
          viewOptions={spanCardViewOptions}
        />
      </ul>
    </div>
  );
};
