import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from '@/components/v1/ui/chart';
import { Skeleton } from '@/components/v1/ui/skeleton';
import { useTenantDetails } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { Bar, BarChart, ResponsiveContainer, XAxis } from 'recharts';

type ActivityMiniChartProps = {
  workflowId: string;
};

const chartConfig: ChartConfig = {
  SUCCEEDED: {
    label: 'Succeeded',
    color: 'rgb(34 197 94 / 0.5)',
  },
  FAILED: {
    label: 'Failed',
    color: 'hsl(var(--destructive))',
  },
};

export const ActivityMiniChart = ({ workflowId }: ActivityMiniChartProps) => {
  const { tenantId } = useTenantDetails();

  // Last 4 days - memoized to prevent query key changes on every render
  const windowStart = useMemo(() => {
    const date = new Date();
    date.setDate(date.getDate() - 3);
    date.setHours(0, 0, 0, 0); // Round to start of day for stable query key
    return date.toISOString();
  }, []);

  const metricsQuery = useQuery({
    ...queries.v1TaskRuns.pointMetrics(tenantId ?? '', {
      createdAfter: windowStart,
      workflow_ids: [workflowId],
    }),
    enabled: !!tenantId,
    staleTime: 5 * 60 * 1000, // 5 minutes
    refetchOnWindowFocus: false,
    placeholderData: (prev) => prev, // Keep previous data while refetching to prevent flicker
  });

  if (metricsQuery.isLoading) {
    return <Skeleton className="h-8 w-24" />;
  }

  const data = metricsQuery.data?.results || [];

  // If no data, show empty state
  if (data.length === 0) {
    return (
      <div className="flex h-8 w-24 items-center justify-center text-xs text-muted-foreground">
        No data
      </div>
    );
  }

  const chartData = data.map((result) => ({
    date: result.time,
    SUCCEEDED: result.SUCCEEDED,
    FAILED: result.FAILED,
  }));

  return (
    <ChartContainer config={chartConfig} className="h-8 w-24 min-h-8">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart
          data={chartData}
          margin={{ left: 0, right: 0, top: 0, bottom: 0 }}
        >
          <XAxis dataKey="date" hide />
          <ChartTooltip
            position={{ y: 40 }}
            content={
              <ChartTooltipContent
                className="w-[140px] font-mono text-xs"
                labelFormatter={(value) =>
                  new Date(value).toLocaleDateString([], {
                    month: 'short',
                    day: 'numeric',
                    hour: '2-digit',
                    minute: '2-digit',
                  })
                }
              />
            }
          />
          <Bar
            dataKey="SUCCEEDED"
            fill={chartConfig.SUCCEEDED.color}
            isAnimationActive={false}
            stackId="a"
          />
          <Bar
            dataKey="FAILED"
            fill={chartConfig.FAILED.color}
            isAnimationActive={false}
            stackId="a"
          />
        </BarChart>
      </ResponsiveContainer>
    </ChartContainer>
  );
};
