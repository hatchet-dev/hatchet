import type { SpanCardViewOptions } from "./SpanCard/SpanCard";
import { SpanCard } from "./SpanCard/SpanCard";
import { findTimeRange } from "./agent-prism-data";
import type { OtelSpanTree } from "./span-tree-type";
import cn from "classnames";
import { type FC, useMemo } from "react";

interface TreeViewProps {
  spanTrees: OtelSpanTree[];
  className?: string;
  selectedSpan?: OtelSpanTree;
  onSpanSelect?: (span: OtelSpanTree) => void;
  expandedSpansIds: string[];
  onExpandSpansIdsChange: (ids: string[]) => void;
  spanCardViewOptions?: SpanCardViewOptions;
}

/**
 * Compute a shared timeline range across all root trees.
 * Each root tree is treated as starting at 0 (its own minStart),
 * and the shared maxEnd is the longest root tree's duration.
 * This makes bar widths comparable across trees without being
 * distorted by gaps between replay sessions.
 */
const useSharedTimeRange = (
  spanTrees: OtelSpanTree[],
): { perTreeMinStart: number[]; sharedMaxDuration: number } => {
  return useMemo(() => {
    const ranges = spanTrees.map((tree) => findTimeRange([tree]));
    const perTreeMinStart = ranges.map((r) => r.minStart);
    const sharedMaxDuration = Math.max(
      ...ranges.map((r) => r.maxEnd - r.minStart),
    );
    return { perTreeMinStart, sharedMaxDuration };
  }, [spanTrees]);
};

export const TreeView: FC<TreeViewProps> = ({
  spanTrees,
  onSpanSelect,
  className = "",
  selectedSpan,
  expandedSpansIds,
  onExpandSpansIdsChange,
  spanCardViewOptions,
}) => {
  const { perTreeMinStart, sharedMaxDuration } = useSharedTimeRange(spanTrees);

  return (
    <div className="w-full min-w-0 px-4">
      <ul
        className={cn(className, "overflow-x-auto pt-2")}
        role="tree"
        aria-label="Hierarchical card list"
      >
        {spanTrees.map((tree, idx) => (
          <SpanCard
            key={tree.spanId}
            data={tree}
            level={0}
            selectedSpan={selectedSpan}
            onSpanSelect={onSpanSelect}
            minStart={perTreeMinStart[idx]}
            maxEnd={perTreeMinStart[idx] + sharedMaxDuration}
            isLastChild={idx === spanTrees.length - 1}
            expandedSpansIds={expandedSpansIds}
            onExpandSpansIdsChange={onExpandSpansIdsChange}
            viewOptions={spanCardViewOptions}
          />
        ))}
      </ul>
    </div>
  );
};
