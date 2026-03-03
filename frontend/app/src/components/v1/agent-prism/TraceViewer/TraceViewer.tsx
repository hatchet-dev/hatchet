import type { TraceRecord, TraceSpan } from "@evilmartians/agent-prism-types";

import {
  filterSpansRecursively,
  flattenSpans,
} from "@evilmartians/agent-prism-data";
import { useCallback, useEffect, useMemo, useState } from "react";

import { type BadgeProps } from "../Badge";
import { useIsMobile, useIsMounted } from "../shared";
import { type SpanCardViewOptions } from "../SpanCard/SpanCard";
import { TraceViewerDesktopLayout } from "./TraceViewerDesktopLayout";
import { TraceViewerMobileLayout } from "./TraceViewerMobileLayout";

export interface TraceViewerData {
  traceRecord: TraceRecord;
  badges?: Array<BadgeProps>;
  spans: TraceSpan[];
  spanCardViewOptions?: SpanCardViewOptions;
}

export interface TraceViewerProps {
  data: Array<TraceViewerData>;
  spanCardViewOptions?: SpanCardViewOptions;
}

export const TraceViewer = ({
  data,
  spanCardViewOptions,
}: TraceViewerProps) => {
  const isMobile = useIsMobile();
  const isMounted = useIsMounted();

  const [selectedSpan, setSelectedSpan] = useState<TraceSpan | undefined>();
  const [searchValue, setSearchValue] = useState("");
  const [traceListExpanded, setTraceListExpanded] = useState(true);

  const [selectedTrace, setSelectedTrace] = useState<
    TraceRecordWithDisplayData | undefined
  >(
    data[0]
      ? {
          ...data[0].traceRecord,
          badges: data[0].badges,
          spanCardViewOptions: data[0].spanCardViewOptions,
        }
      : undefined,
  );
  const [selectedTraceSpans, setSelectedTraceSpans] = useState<TraceSpan[]>(
    data[0]?.spans || [],
  );

  const traceRecords: TraceRecordWithDisplayData[] = useMemo(() => {
    return data.map((item) => ({
      ...item.traceRecord,
      badges: item.badges,
      spanCardViewOptions: item.spanCardViewOptions,
    }));
  }, [data]);

  const filteredSpans = useMemo(() => {
    if (!searchValue.trim()) {
      return selectedTraceSpans;
    }
    return filterSpansRecursively(selectedTraceSpans, searchValue);
  }, [selectedTraceSpans, searchValue]);

  const allIds = useMemo(() => {
    return flattenSpans(selectedTraceSpans).map((span) => span.id);
  }, [selectedTraceSpans]);

  const [expandedSpansIds, setExpandedSpansIds] = useState<string[]>(allIds);

  useEffect(() => {
    setExpandedSpansIds(allIds);
  }, [allIds]);

  useEffect(() => {
    if (!isMounted || isMobile) return;

    if (selectedTraceSpans.length > 0 && !selectedSpan) {
      setSelectedSpan(selectedTraceSpans[0]);
    }
  }, [selectedTraceSpans, isMobile, isMounted, selectedSpan]);

  const handleExpandAll = useCallback(() => {
    setExpandedSpansIds(allIds);
  }, [allIds]);

  const handleCollapseAll = useCallback(() => {
    setExpandedSpansIds([]);
  }, []);

  const handleTraceSelect = useCallback(
    (trace: TraceRecord) => {
      setSelectedSpan(undefined);
      setExpandedSpansIds([]);
      setSelectedTrace(trace);
      setSelectedTraceSpans(
        data.find((item) => item.traceRecord.id === trace.id)?.spans ?? [],
      );
    },
    [data],
  );

  const handleClearTraceSelection = useCallback(() => {
    setSelectedTrace(undefined);
    setSelectedTraceSpans([]);
    setSelectedSpan(undefined);
    setExpandedSpansIds([]);
  }, []);

  const props: TraceViewerLayoutProps = {
    traceRecords,
    traceListExpanded,
    setTraceListExpanded,
    selectedTrace,
    selectedTraceId: selectedTrace?.id,
    selectedSpan,
    setSelectedSpan,
    searchValue,
    setSearchValue,
    filteredSpans,
    expandedSpansIds,
    setExpandedSpansIds,
    handleExpandAll,
    handleCollapseAll,
    handleTraceSelect,
    spanCardViewOptions:
      spanCardViewOptions || selectedTrace?.spanCardViewOptions,
    onClearTraceSelection: handleClearTraceSelection,
  };

  return (
    <div className="h-[calc(100vh-50px)]">
      <div className="hidden h-full lg:block">
        <TraceViewerDesktopLayout {...props} />
      </div>
      <div className="h-full lg:hidden">
        <TraceViewerMobileLayout {...props} />
      </div>
    </div>
  );
};

export interface TraceRecordWithDisplayData extends TraceRecord {
  spanCardViewOptions?: SpanCardViewOptions;
  badges?: BadgeProps[];
}

export interface TraceViewerLayoutProps {
  traceRecords: TraceRecordWithDisplayData[];
  traceListExpanded: boolean;
  setTraceListExpanded: (expanded: boolean) => void;
  selectedTrace: TraceRecordWithDisplayData | undefined;
  selectedTraceId?: string;
  selectedSpan: TraceSpan | undefined;
  setSelectedSpan: (span: TraceSpan | undefined) => void;
  searchValue: string;
  setSearchValue: (value: string) => void;
  filteredSpans: TraceSpan[];
  expandedSpansIds: string[];
  setExpandedSpansIds: (ids: string[]) => void;
  handleExpandAll: () => void;
  handleCollapseAll: () => void;
  handleTraceSelect: (trace: TraceRecord) => void;
  spanCardViewOptions?: SpanCardViewOptions;
  onClearTraceSelection: () => void;
}
