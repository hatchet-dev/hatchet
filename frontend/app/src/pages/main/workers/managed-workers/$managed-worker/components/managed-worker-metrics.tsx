import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import {
  ManagedWorker,
  SampleStream,
} from '@/lib/api/generated/cloud/data-contracts';
import { Loading } from '@/components/ui/loading';
import AreaChart, {
  MetricValue,
  format2Dec,
  formatPercentTooltip,
} from '@/components/molecules/brush-chart/area-chart';
import { useMemo, useState } from 'react';
import { useParentSize } from '@visx/responsive';
import { Separator } from '@/components/ui/separator';
import { GetCloudMetricsQuery } from '@/lib/api/queries';
import { DateTimePicker } from '@/components/molecules/time-picker/date-time-picker';
import { Button } from '@/components/ui/button';
import { ArrowPathIcon } from '@heroicons/react/24/outline';

export function ManagedWorkerMetrics({
  managedWorker,
}: {
  managedWorker: ManagedWorker;
}) {
  const [beforeInput, setBeforeInput] = useState<Date | undefined>();
  const [afterInput, setAfterInput] = useState<Date | undefined>();
  const [queryParams, setQueryParams] = useState<GetCloudMetricsQuery>({
    // default after is 1 day
    after: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
  });
  const [rotate, setRotate] = useState(false);

  const getCpuMetricsQuery = useQuery({
    ...queries.cloud.getManagedWorkerCpuMetrics(
      managedWorker?.metadata.id || '',
      queryParams,
    ),
    enabled: !!managedWorker,
    refetchInterval: () => {
      return 5000;
    },
  });

  const getMemoryMetricsQuery = useQuery({
    ...queries.cloud.getManagedWorkerMemoryMetrics(
      managedWorker?.metadata.id || '',
      queryParams,
    ),
    enabled: !!managedWorker,
    refetchInterval: () => {
      return 5000;
    },
  });

  const getDiskMetricsQuery = useQuery({
    ...queries.cloud.getManagedWorkerDiskMetrics(
      managedWorker?.metadata.id || '',
      queryParams,
    ),
    enabled: !!managedWorker,
    refetchInterval: () => {
      return 5000;
    },
  });

  if (
    getCpuMetricsQuery.isLoading ||
    getMemoryMetricsQuery.isLoading ||
    getDiskMetricsQuery.isLoading
  ) {
    return <Loading />;
  }

  const refreshMetrics = () => {
    setQueryParams({
      after: afterInput?.toISOString(),
      before: beforeInput?.toISOString(),
    });
    setRotate(!rotate);
  };

  const datesMatchSearch =
    beforeInput?.toISOString() === queryParams?.before &&
    afterInput?.toISOString() === queryParams?.after;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-row justify-between items-center mt-4">
        <h3 className="text-xl font-bold leading-tight text-foreground">
          Metrics
        </h3>
        <div className="flex flex-row gap-4">
          <DateTimePicker
            date={afterInput}
            setDate={setAfterInput}
            label="After"
          />
          <DateTimePicker
            date={beforeInput}
            setDate={setBeforeInput}
            label="Before"
          />
          <Button
            key="refresh"
            className="h-8 px-2 lg:px-3"
            size="sm"
            onClick={refreshMetrics}
            variant={datesMatchSearch ? 'outline' : 'default'}
            aria-label="Refresh logs"
          >
            <ArrowPathIcon
              className={`h-4 w-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
            />
          </Button>
        </div>
      </div>
      <Separator />
      <h4 className="text-lg font-bold leading-tight text-foreground mt-2 ml-4">
        CPU
      </h4>
      <Separator />
      {getCpuMetricsQuery.data?.length === 0 && (
        <MetricsPlaceholder
          start={afterInput || new Date(Date.now() - 24 * 60 * 60 * 1000)}
          end={beforeInput || new Date()}
        />
      )}
      {getCpuMetricsQuery.data?.map((d, i) => {
        return (
          <MetricsChart
            key={i}
            sample={d}
            yLabel="CPU Usage (%)"
            tooltipFormat={formatPercentTooltip}
          />
        );
      })}
      <h4 className="text-lg font-bold leading-tight text-foreground mt-16 ml-4">
        Memory
      </h4>
      <Separator />
      {getMemoryMetricsQuery.data?.map((d, i) => {
        return (
          <MetricsChart
            key={i}
            sample={d}
            normalizer={(d) => {
              return d / (1000 * 1000);
            }}
            yLabel="Memory (MB)"
            tooltipFormat={(d) => {
              return format2Dec(d) + ' MB';
            }}
          />
        );
      })}
      <h4 className="text-lg font-bold leading-tight text-foreground mt-16 ml-4">
        Disk
      </h4>
      <Separator />
      {getDiskMetricsQuery.data?.map((d, i) => {
        return (
          <MetricsChart
            key={i}
            sample={d}
            normalizer={(d) => {
              return d / (1000 * 1000);
            }}
            yLabel="Disk (MB)"
            tooltipFormat={(d) => {
              return format2Dec(d) + ' MB';
            }}
          />
        );
      })}
    </div>
  );
}

type MetricsChartProps = {
  sample: SampleStream;
  normalizer?: (value: number) => number;
  yLabel: string;
  tooltipFormat?: (d: number) => string;
};

function MetricsChart({
  sample,
  normalizer,
  yLabel,
  tooltipFormat,
}: MetricsChartProps) {
  const { parentRef, width, height } = useParentSize({ debounceTime: 150 });

  const values: MetricValue[] = useMemo(
    () =>
      sample.values?.map((v) => {
        return {
          date: new Date(v[0] * 1000),
          value: normalizer ? normalizer(parseFloat(v[1])) : parseFloat(v[1]),
        };
      }) || [],
    [sample, normalizer],
  );

  return (
    <div
      ref={parentRef}
      className="w-full max-h-[25rem] min-h-[25rem] ml-8 px-14"
    >
      <AreaChart
        hideBottomAxis={false}
        data={values}
        width={width}
        height={height}
        yLabel={yLabel}
        tooltipFormat={tooltipFormat}
      />
    </div>
  );
}

function MetricsPlaceholder({ start, end }: { start: Date; end: Date }) {
  const { parentRef, width, height } = useParentSize({ debounceTime: 150 });

  return (
    <div
      ref={parentRef}
      className="w-full max-h-[25rem] min-h-[25rem] ml-8 px-14"
    >
      <AreaChart
        hideBottomAxis={true}
        hideLeftAxis={true}
        data={[]}
        width={width}
        height={height}
        yDomain={[0, 100]}
        xDomain={[start, end]}
        centerText="No data available for the selected time range."
      />
    </div>
  );
}
