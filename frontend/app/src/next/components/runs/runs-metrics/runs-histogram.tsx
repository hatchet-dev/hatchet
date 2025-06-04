import { Skeleton } from '@/next/components/ui/skeleton';
import { DataPoint, ZoomableChart } from '@/next/components/ui/charts/zoomable';
import { useRuns } from '@/next/hooks/use-runs';

function GetWorkflowChart() {
  const { histogram, timeRange } = useRuns();

  if (histogram.isLoading) {
    return <Skeleton className="w-full h-36" />;
  }

  return (
    <div className="">
      <ZoomableChart
        kind="bar"
        data={
          histogram.data?.results?.map(
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
          timeRange.setTimeFilter({
            startTime: start,
            endTime: end,
          });
        }}
        showYAxis={false}
      />
    </div>
  );
}

export default GetWorkflowChart;
