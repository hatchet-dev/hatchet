import { SpanDetail, GroupDetail } from './span-detail';
import { TraceMinimap } from './trace-minimap';
import {
  TraceTimeline,
  LABEL_WIDTH,
  type VisibleRange,
  type SpanGroupInfo,
} from './trace-timeline';
import { findTimeRange } from '@/components/v1/agent-prism/agent-prism-data';
import { convertOtelSpansToOtelSpanTree } from '@/components/v1/agent-prism/convert-otel-spans-to-agent-prism-span-tree';
import type {
  OtelSpanTree,
  RelevantOpenTelemetrySpanProperties,
} from '@/components/v1/agent-prism/span-tree-type';
import { useCallback, useMemo, useState } from 'react';

type Selection =
  | { kind: 'span'; span: OtelSpanTree }
  | { kind: 'group'; group: SpanGroupInfo };

export function TaskRunTrace({
  spans,
}: {
  spans: [
    RelevantOpenTelemetrySpanProperties,
    ...RelevantOpenTelemetrySpanProperties[],
  ];
  onTaskRunClick?: (taskRunId: string) => void;
}) {
  const spanTrees = useMemo(
    () => convertOtelSpansToOtelSpanTree(spans),
    [spans],
  );

  const { minStart, maxEnd } = useMemo(
    () => findTimeRange(spanTrees),
    [spanTrees],
  );

  const [expandedSpansIds, setExpandedSpansIds] = useState<string[]>(() =>
    spanTrees.map((s) => s.spanId),
  );

  const [groupVisibleCounts, setGroupVisibleCounts] = useState<
    Record<string, number>
  >({});

  const [selection, setSelection] = useState<Selection | undefined>();
  const [visibleRange, setVisibleRange] = useState<VisibleRange>({
    startPct: 0,
    endPct: 1,
  });

  const handleSpanSelect = useCallback((span: OtelSpanTree) => {
    setSelection((prev) =>
      prev?.kind === 'span' && prev.span.spanId === span.spanId
        ? undefined
        : { kind: 'span', span },
    );
  }, []);

  const handleGroupSelect = useCallback((group: SpanGroupInfo) => {
    setSelection((prev) =>
      prev?.kind === 'group' && prev.group.groupId === group.groupId
        ? undefined
        : { kind: 'group', group },
    );
  }, []);

  const handleDetailClose = useCallback(() => {
    setSelection(undefined);
  }, []);

  const handleShowMore = useCallback(
    (groupId: string, newVisibleCount: number) => {
      setGroupVisibleCounts((prev) => ({
        ...prev,
        [groupId]: newVisibleCount,
      }));
    },
    [],
  );

  if (spanTrees.length === 0) {
    return (
      <div className="py-4 text-sm text-muted-foreground">
        No trace found for this task run. To collect traces, use the{' '}
        <code className="rounded bg-muted px-1 py-0.5 text-xs">
          HatchetInstrumentor
        </code>{' '}
        in your SDK.
      </div>
    );
  }

  return (
    <div className="my-4 flex min-w-0 flex-col gap-4 overflow-hidden">
      <div className="flex min-w-0">
        <div className="shrink-0" style={{ width: LABEL_WIDTH }} />
        <div className="min-w-0 flex-1 pr-10">
          <TraceMinimap
            spanTrees={spanTrees}
            minMs={minStart}
            maxMs={maxEnd}
            visibleRange={visibleRange}
            onRangeChange={setVisibleRange}
          />
        </div>
      </div>
      <TraceTimeline
        spanTrees={spanTrees}
        expandedSpanIds={expandedSpansIds}
        onExpandChange={setExpandedSpansIds}
        groupVisibleCounts={groupVisibleCounts}
        onShowMore={handleShowMore}
        selectedSpan={selection?.kind === 'span' ? selection.span : undefined}
        selectedGroupId={
          selection?.kind === 'group' ? selection.group.groupId : undefined
        }
        onSpanSelect={handleSpanSelect}
        onGroupSelect={handleGroupSelect}
        visibleRange={visibleRange}
      />
      {selection?.kind === 'span' && (
        <SpanDetail span={selection.span} onClose={handleDetailClose} />
      )}
      {selection?.kind === 'group' && (
        <GroupDetail group={selection.group} onClose={handleDetailClose} />
      )}
    </div>
  );
}
