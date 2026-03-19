import { RunsTable } from './components/runs-table';
import { RunsProvider } from './hooks/runs-provider';
import { useEffect } from 'react';

export default function Tasks() {
  useEffect(() => {
    console.log('[WorkflowRunsV1] mount', {
      path: window.location.pathname,
    });
  }, []);

  return (
    <div className="size-full flex-grow">
      <RunsProvider tableKey="workflow-runs-main">
        <RunsTable />
      </RunsProvider>
    </div>
  );
}
