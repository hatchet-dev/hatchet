import { RunsTable } from './components/runs-table';
import { RunsProvider } from './hooks/runs-provider';

export default function Tasks() {
  return (
    <div className="flex-grow size-full">
      <RunsProvider
        tableKey="workflow-runs-main"
        display={{
          showMetrics: true,
          showCounts: true,
          showDateFilter: true,
          showTriggerRunButton: true,
          showColumnToggle: true,
          showCancelAndReplayButtons: true,
        }}
        runFilters={{}}
        initColumnVisibility={{}}
        filterVisibility={{}}
      >
        <RunsTable />
      </RunsProvider>
    </div>
  );
}
