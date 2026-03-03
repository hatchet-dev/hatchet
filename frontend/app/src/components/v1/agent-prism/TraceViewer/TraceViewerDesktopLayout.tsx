import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";

import { DetailsView } from "../DetailsView/DetailsView";
import { TraceList } from "../TraceList/TraceList";
import { type TraceViewerLayoutProps } from "./TraceViewer";
import { TraceViewerPlaceholder } from "./TraceViewerPlaceholder";
import { TraceViewerTreeViewContainer } from "./TraceViewerTreeViewContainer";

export const TraceViewerDesktopLayout = ({
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
}: TraceViewerLayoutProps) => {
  const actualSelectedTrace =
    traceRecords.find((t) => t.id === selectedTraceId) || selectedTrace;

  return (
    <PanelGroup direction="horizontal" className="h-full">
      <Panel
        id="trace-list"
        defaultSize={traceListExpanded ? 20 : 2}
        minSize={traceListExpanded ? 15 : 2}
        maxSize={traceListExpanded ? 40 : 2}
        collapsible={false}
        className="flex h-full min-h-0 flex-col overflow-hidden"
      >
        <TraceList
          traces={traceRecords}
          expanded={traceListExpanded}
          onExpandStateChange={setTraceListExpanded}
          onTraceSelect={handleTraceSelect}
          selectedTrace={actualSelectedTrace}
        />
      </Panel>

      <PanelResizeHandle />

      {selectedTrace ? (
        <Panel
          id="tree-view"
          minSize={30}
          className="flex h-full flex-col gap-y-2 overflow-hidden"
        >
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
        </Panel>
      ) : (
        <Panel
          id="tree-view"
          minSize={30}
          className="flex h-full items-center justify-center"
        >
          <TraceViewerPlaceholder title="Select a trace to see the details" />
        </Panel>
      )}

      <PanelResizeHandle />

      <Panel
        id="details-view"
        defaultSize={30}
        minSize={20}
        maxSize={50}
        className="h-full overflow-hidden"
      >
        {selectedSpan ? (
          <DetailsView data={selectedSpan} />
        ) : (
          <TraceViewerPlaceholder title="Select a span to see the details" />
        )}
      </Panel>
    </PanelGroup>
  );
};
