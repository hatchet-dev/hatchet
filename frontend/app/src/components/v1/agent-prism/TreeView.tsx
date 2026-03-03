import type { TraceSpan } from "@evilmartians/agent-prism-types";

import { flattenSpans, findTimeRange } from "@evilmartians/agent-prism-data";
import cn from "classnames";
import { type FC } from "react";

import type { SpanCardViewOptions } from "./SpanCard/SpanCard";

import { BrandLogo } from "./BrandLogo";
import { SpanCard } from "./SpanCard/SpanCard";

interface TreeViewProps {
  spans: TraceSpan[];
  className?: string;
  selectedSpan?: TraceSpan;
  onSpanSelect?: (span: TraceSpan) => void;
  expandedSpansIds: string[];
  onExpandSpansIdsChange: (ids: string[]) => void;
  spanCardViewOptions?: SpanCardViewOptions;
}

export const TreeView: FC<TreeViewProps> = ({
  spans,
  onSpanSelect,
  className = "",
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
        className={cn(className, "overflow-x-auto pt-2")}
        role="tree"
        aria-label="Hierarchical card list"
      >
        {spans.map((span, idx) => {
          const brand = span.metadata?.brand as { type: string } | undefined;

          return (
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
              avatar={
                brand
                  ? {
                      children: <BrandLogo brand={brand.type} />,
                      size: "4",
                      rounded: "sm",
                      category: span.type,
                    }
                  : undefined
              }
            />
          );
        })}
      </ul>
    </div>
  );
};
