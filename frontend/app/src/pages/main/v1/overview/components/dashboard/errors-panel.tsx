import { truncateError } from './dashboard-metrics';
import { PanelCard, PanelState } from './panel-card';
import RelativeDate from '@/components/v1/molecules/relative-date';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/v1/ui/table';
import useControlPlane from '@/hooks/use-control-plane';
import { queries, V1TaskStatus, type V1TaskSummary } from '@/lib/api';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { AlertTriangle } from 'lucide-react';

const REFETCH_MS = 20000;
const ERROR_LIMIT = 5;

export function ErrorsPanel({
  tenantId,
  since,
}: {
  tenantId: string;
  since: string;
}) {
  const { isSelfHosted } = useControlPlane();

  const errorsQuery = useQuery({
    ...queries.v1WorkflowRuns.list(
      tenantId,
      {
        offset: 0,
        limit: ERROR_LIMIT,
        statuses: [V1TaskStatus.FAILED],
        since,
        only_tasks: false,
        include_payloads: false,
      },
      isSelfHosted,
    ),
    refetchInterval: REFETCH_MS,
    placeholderData: (prev) => prev,
  });

  const data = errorsQuery.data;
  const timedOut = data === 'timeout';
  const rows: V1TaskSummary[] =
    data && data !== 'timeout' ? (data.rows ?? []) : [];

  return (
    <PanelCard
      icon={<AlertTriangle className="size-4" />}
      title="Recent errors"
      subtitle="Failed runs, last 24 hours"
    >
      <PanelState
        loading={errorsQuery.isLoading}
        error={errorsQuery.isError || timedOut}
        onRetry={() => errorsQuery.refetch()}
        isEmpty={rows.length === 0}
        emptyText="No failures in the last 24 hours."
      >
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="whitespace-nowrap">When</TableHead>
                <TableHead className="whitespace-nowrap">Task</TableHead>
                <TableHead>Error</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row.metadata.id}>
                  <TableCell className="whitespace-nowrap align-top text-xs text-muted-foreground">
                    <RelativeDate date={row.createdAt} />
                  </TableCell>
                  <TableCell className="align-top">
                    <Link
                      to={appRoutes.tenantRunRoute.to}
                      params={{ tenant: tenantId, run: row.metadata.id }}
                      className="whitespace-nowrap text-sm hover:underline"
                    >
                      {row.displayName}
                    </Link>
                  </TableCell>
                  <TableCell className="align-top">
                    <div className="flex flex-col gap-1">
                      <span className="font-mono text-xs text-muted-foreground">
                        {truncateError(row.errorMessage) || 'Unknown error'}
                      </span>
                      <Link
                        to={appRoutes.tenantRunRoute.to}
                        params={{ tenant: tenantId, run: row.metadata.id }}
                        className="text-xs text-primary/70 hover:text-primary hover:underline"
                      >
                        View run
                      </Link>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </PanelState>
    </PanelCard>
  );
}
