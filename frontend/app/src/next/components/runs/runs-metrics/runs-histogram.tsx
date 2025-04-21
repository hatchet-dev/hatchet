import { useQuery } from '@tanstack/react-query';
import { Skeleton } from '@/next/components/ui/skeleton';
import { DataPoint } from '@/next/components/ui/charts/zoomable';
import { ZoomableChart } from '@/next/components/ui/charts/zoomable';
import { queries } from '@/lib/api/queries';

interface WorkflowChartProps {
  tenantId: string;
  createdAfter?: string;
  finishedBefore?: string;
  refetchInterval?: number;
  zoom: (startTime: string, endTime: string) => void;
}

const GetWorkflowChart = ({
  tenantId,
  createdAfter,
  finishedBefore,
  refetchInterval,
  zoom,
}: WorkflowChartProps) => {
  const workflowRunEventsMetricsQuery = useQuery({
    ...queries.cloud.workflowRunMetrics(tenantId, {
      createdAfter,
      finishedBefore,
    }),
    placeholderData: (prev: any) => prev,
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
        zoom={zoom}
        showYAxis={false}
      />
    </div>
  );
};

export default GetWorkflowChart;
