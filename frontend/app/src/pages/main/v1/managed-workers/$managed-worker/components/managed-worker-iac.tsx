import { ManagedWorkerIaCLogs } from './managed-worker-iac-logs';

export function ManagedWorkerIaC({
  managedWorkerId,
  deployKey,
}: {
  managedWorkerId: string;
  deployKey: string;
}) {
  return (
    <div className="flex flex-col gap-4 w-full">
      <h4 className="text-lg font-semibold text-foreground">IaC Logs</h4>
      <ManagedWorkerIaCLogs
        managedWorkerId={managedWorkerId}
        deployKey={deployKey}
      />
    </div>
  );
}
