import { TaskRunsTable } from './components/task-runs-table';

export default function Tasks() {
  return (
    <div className="flex-grow size-full">
      <TaskRunsTable showMetrics={true} />
    </div>
  );
}
