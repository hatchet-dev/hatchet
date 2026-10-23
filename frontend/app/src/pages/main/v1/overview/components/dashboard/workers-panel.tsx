import { PanelCard, PanelState } from './panel-card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/v1/ui/table';
import { queries, WorkerStatus, type Worker } from '@/lib/api';
import { SdkInfo } from '@/pages/main/v1/workers/components/sdk-info';
import { WorkerStatusBadge } from '@/pages/main/v1/workers/components/worker-columns';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { RiStackLine } from 'react-icons/ri';

const REFETCH_MS = 20000;
const WORKER_LIMIT = 6;

// Aggregate available / limit slots across every slot type of a worker, so the
// panel can show a single compact "free" figure per worker.
function slotSummary(worker: Worker): { available: number; limit: number } {
  const slotConfig = worker.slotConfig ?? {};
  let available = 0;
  let limit = 0;
  for (const cfg of Object.values(slotConfig)) {
    const cfgLimit = cfg?.limit ?? 0;
    limit += cfgLimit;
    available += cfg?.available ?? cfgLimit;
  }
  return { available, limit };
}

export function WorkersPanel({ tenantId }: { tenantId: string }) {
  const workersQuery = useQuery({
    ...queries.workers.list(tenantId),
    refetchInterval: REFETCH_MS,
    placeholderData: (prev) => prev,
  });

  // Only active workers are relevant here; the worker list also returns
  // recently-seen inactive/paused ones.
  const activeWorkers = (workersQuery.data?.rows ?? []).filter(
    (worker) => worker.status === WorkerStatus.ACTIVE,
  );
  const rows = activeWorkers.slice(0, WORKER_LIMIT);
  const hiddenCount = activeWorkers.length - rows.length;

  return (
    <PanelCard
      icon={<RiStackLine className="size-4" />}
      title="Workers"
      subtitle="Connected"
    >
      <PanelState
        loading={workersQuery.isLoading}
        error={workersQuery.isError}
        onRetry={() => workersQuery.refetch()}
        isEmpty={rows.length === 0}
        emptyText="No workers connected. Start one to run tasks."
      >
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="whitespace-nowrap">Worker</TableHead>
                <TableHead className="whitespace-nowrap">Status</TableHead>
                <TableHead className="whitespace-nowrap">Slots</TableHead>
                <TableHead className="whitespace-nowrap">SDK</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((worker) => {
                const slots = slotSummary(worker);
                return (
                  <TableRow key={worker.metadata.id}>
                    <TableCell className="align-middle">
                      <Link
                        to={appRoutes.tenantWorkerRoute.to}
                        params={{
                          tenant: tenantId,
                          worker: worker.metadata.id,
                        }}
                        className="whitespace-nowrap text-xs hover:underline"
                      >
                        {worker.webhookUrl || worker.name}
                      </Link>
                    </TableCell>
                    <TableCell className="align-middle">
                      <WorkerStatusBadge status={worker.status} />
                    </TableCell>
                    <TableCell className="whitespace-nowrap align-middle text-xs text-muted-foreground">
                      {slots.limit > 0
                        ? `${slots.available} / ${slots.limit} free`
                        : 'No slots'}
                    </TableCell>
                    <TableCell className="align-middle text-xs text-muted-foreground">
                      <SdkInfo runtimeInfo={worker.runtimeInfo} />
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
        <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
          <span>
            {hiddenCount > 0
              ? `Showing ${rows.length} of ${activeWorkers.length} active workers`
              : `${activeWorkers.length} active ${
                  activeWorkers.length === 1 ? 'worker' : 'workers'
                }`}
          </span>
          <Link
            to={appRoutes.tenantWorkersRoute.to}
            params={{ tenant: tenantId }}
            className="hover:text-foreground hover:underline"
          >
            View all workers
          </Link>
        </div>
      </PanelState>
    </PanelCard>
  );
}
