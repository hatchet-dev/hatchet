import { RunsTable } from './components/runs-table';

export default function Tasks() {
  return (
    <div className="flex-grow size-full">
      <RunsTable tableKey="workflow-runs-main" showMetrics={true} />
    </div>
  );
}
