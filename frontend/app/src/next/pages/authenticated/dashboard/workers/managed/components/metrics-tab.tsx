import { useManagedComputeDetail } from '@/next/hooks/use-managed-compute-detail';
import { FC, useEffect } from 'react';
import { Separator } from '@/next/components/ui/separator';
import { Spinner } from '@/next/components/ui/spinner';
import { Matrix } from '@/lib/api/generated/cloud/data-contracts';
import {
  ZoomableChart,
  DataPoint,
} from '@/components/molecules/charts/zoomable';
import {
  TimeFilter,
  TimeFilterGroup,
  TogglePause,
} from '@/next/components/ui/filters/time-filter';
import {
  TimeFilterProvider,
  useTimeFilters,
} from '@/next/hooks/utils/use-time-filters';

function MetricsContent() {
  const {
    data: managedWorker,
    metrics,
    setMetricsQuery,
  } = useManagedComputeDetail();
  const { filters, setTimeFilter, pause } = useTimeFilters();

  // Update metrics query when time range changes
  useEffect(() => {
    const newQuery = {
      after: filters.startTime,
      before: filters.endTime,
    };
    setMetricsQuery(newQuery);
  }, [filters.startTime, filters.endTime, setMetricsQuery]);

  // Handle zoom functionality
  const handleZoom = (startTime: string, endTime: string) => {
    // Pause the time filter to enable custom range mode
    pause();
    // Set the custom time range based on zoom selection
    setTimeFilter({
      startTime,
      endTime,
    });
  };

  if (
    !managedWorker ||
    metrics?.cpu?.isLoading ||
    metrics?.memory?.isLoading ||
    metrics?.disk?.isLoading ||
    !metrics?.cpu?.data ||
    !metrics?.memory?.data ||
    !metrics?.disk?.data
  ) {
    return <Spinner />;
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-row justify-between items-center">
        <h3 className="text-xl font-bold leading-tight text-foreground">
          Metrics
        </h3>
        <TimeFilterGroup className="justify-end">
          <TimeFilter />
          <TogglePause />
        </TimeFilterGroup>
      </div>
      <Separator />

      <h4 className="text-lg font-bold leading-tight text-foreground mt-2 ml-4">
        CPU
      </h4>
      <Separator />
      <ZoomableChart
        className="max-h-[25rem] min-h-[25rem]"
        data={transformToDataPoints(metrics.cpu.data)}
        kind="line"
        zoom={handleZoom}
        showYAxis={true}
      />

      <h4 className="text-lg font-bold leading-tight text-foreground mt-16 ml-4">
        Memory
      </h4>
      <Separator />
      <ZoomableChart
        className="max-h-[25rem] min-h-[25rem]"
        data={transformToDataPoints(
          metrics.memory.data,
          (d) => d / (1000 * 1000),
        )}
        kind="line"
        zoom={handleZoom}
        showYAxis={true}
      />

      <h4 className="text-lg font-bold leading-tight text-foreground mt-16 ml-4">
        Disk
      </h4>
      <Separator />
      <ZoomableChart
        className="max-h-[25rem] min-h-[25rem]"
        data={transformToDataPoints(
          metrics.disk.data,
          (d) => d / (1000 * 1000),
        )}
        kind="line"
        zoom={handleZoom}
        showYAxis={true}
      />
    </div>
  );
}

export const MetricsTab: FC = () => {
  return (
    <TimeFilterProvider
      initialTimeRange={{
        activePreset: '1h',
      }}
    >
      <MetricsContent />
    </TimeFilterProvider>
  );
};

function transformToDataPoints(
  matrix: Matrix,
  normalizer?: (n: number) => number,
): DataPoint<string>[] {
  const dataPointsMap: Record<string, any> = {};

  matrix.forEach((sampleStream) => {
    // if we have instance or region, use that as the metricLabel
    let metricLabel = Object.values(sampleStream.metric || {}).join('-');

    if (sampleStream.metric?.instance && sampleStream.metric?.region) {
      metricLabel = `[${sampleStream.metric.region}] ${sampleStream.metric.instance}`;
    }

    (sampleStream.values || []).forEach(([timestamp, value]) => {
      const isoDate = new Date(timestamp * 1000).toISOString();

      if (!dataPointsMap[isoDate]) {
        dataPointsMap[isoDate] = {
          date: isoDate,
        };
      }

      let val = parseFloat(value);
      val = normalizer ? normalizer(val) : val;
      dataPointsMap[isoDate][metricLabel] = val;
    });
  });

  return Object.values(dataPointsMap) as DataPoint<string>[];
}
