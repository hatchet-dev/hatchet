import { useQuery } from '@tanstack/react-query';
import { Skeleton } from '@/next/components/ui/skeleton';
import { DataPoint } from '@/next/components/ui/charts/zoomable';
import { ZoomableChart } from '@/next/components/ui/charts/zoomable';
import { queries } from '@/next/lib/api/queries';
import { RunsFilters } from '@/next/hooks/use-runs';
import { useFilters } from '@/next/hooks/use-filters';
import useTenant from '@/next/hooks/use-tenant';
import invariant from 'tiny-invariant';

interface WorkflowChartProps {
  refetchInterval?: number;
}

const GetWorkflowChart = ({ refetchInterval }: WorkflowChartProps) => {
  const { tenant } = useTenant();
  const { filters, setFilters } = useFilters<RunsFilters>();

  invariant(tenant, 'Tenant is required'); // TODO: REMOVE

  const workflowRunEventsMetricsQuery = useQuery({
    ...queries.v1TaskRuns.pointMetrics(tenant?.metadata.id, {
      createdAfter: filters.createdAfter,
      finishedBefore: filters.createdBefore, // TODO: THIS ISN'T CORRECT
    }),
    placeholderData: (prev: any) => prev,
    enabled: !!tenant?.metadata.id,
    refetchInterval,
  });

  if (workflowRunEventsMetricsQuery.isLoading) {
    return <Skeleton className="w-full h-36" />;
  }

  return (
    <div className="">
      <ZoomableChart
        kind="bar"
        data={
          workflowRunEventsMetricsQuery.data?.results?.map(
            (result: any): DataPoint<'SUCCEEDED' | 'FAILED'> => ({
              date: result.time,
              SUCCEEDED: result.SUCCEEDED,
              FAILED: result.FAILED,
            }),
          ) || []
        }
        colors={{
          SUCCEEDED: 'rgb(34 197 94 / 0.5)',
          FAILED: 'hsl(var(--destructive))',
        }}
        zoom={(start, end) => {
          setFilters({
            createdAfter: start,
            createdBefore: end,
          });
        }}
        showYAxis={false}
      />
    </div>
  );
};

export default GetWorkflowChart;
