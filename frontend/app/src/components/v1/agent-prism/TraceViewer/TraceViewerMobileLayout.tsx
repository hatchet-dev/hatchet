import { ArrowLeft } from "lucide-react";

import { Button } from "../Button";
import { DetailsView } from "../DetailsView/DetailsView";
import { TraceList } from "../TraceList/TraceList";
import { type TraceViewerLayoutProps } from "../TraceViewer/TraceViewer";
import { TraceViewerTreeViewContainer } from "./TraceViewerTreeViewContainer";

export const TraceViewerMobileLayout = ({
  traceRecords,
  traceListExpanded,
  setTraceListExpanded,
  selectedTrace,
  selectedTraceId,
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
  spanCardViewOptions,
  onClearTraceSelection,
}: TraceViewerLayoutProps) => {
  if (
    selectedTrace &&
    selectedTraceId &&
    filteredSpans.length > 0 &&
    selectedSpan
  ) {
    return (
      <div className="flex h-full flex-col gap-4 overflow-y-auto">
        <Button
          onClick={() => {
            setSelectedSpan(undefined);
          }}
          iconStart={<ArrowLeft className="size-3" />}
          variant="ghost"
          className="self-start"
        >
          Tree View
        </Button>
        <DetailsView data={selectedSpan} />
      </div>
    );
  }

  if (
    selectedTrace &&
    selectedTraceId &&
    filteredSpans.length > 0 &&
    !selectedSpan
  ) {
    return (
      <div className="flex h-full flex-col gap-4">
        <div className="shrink-0">
          <Button
            onClick={() => {
              if (onClearTraceSelection) {
                onClearTraceSelection();
              }
            }}
            iconStart={<ArrowLeft className="size-3" />}
            variant="ghost"
            className="self-start"
          >
            Traces list
          </Button>
        </div>

        <TraceViewerTreeViewContainer
          searchValue={searchValue}
          setSearchValue={setSearchValue}
          handleExpandAll={handleExpandAll}
          handleCollapseAll={handleCollapseAll}
          filteredSpans={filteredSpans}
          selectedSpan={selectedSpan}
          setSelectedSpan={setSelectedSpan}
          expandedSpansIds={expandedSpansIds}
          setExpandedSpansIds={setExpandedSpansIds}
          spanCardViewOptions={spanCardViewOptions}
          selectedTrace={selectedTrace}
        />
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto">
      <TraceList
        traces={traceRecords}
        expanded={traceListExpanded}
        onExpandStateChange={setTraceListExpanded}
        onTraceSelect={handleTraceSelect}
        selectedTrace={traceRecords.find((t) => t.id === selectedTraceId)}
      />
    </div>
  );
};
