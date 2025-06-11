import { useManagedComputeDetail } from '@/next/hooks/use-managed-compute-detail';
import { FC, useEffect, useState } from 'react';
import { DateTimePicker } from '@/components/molecules/time-picker/date-time-picker';
import { Button } from '@/next/components/ui/button';
import { XCircleIcon } from '@heroicons/react/24/outline';
import { Separator } from '@/next/components/ui/separator';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/next/components/ui/select';
import { Spinner } from '@/next/components/ui/spinner';
import { Matrix } from '@/lib/api/generated/cloud/data-contracts';
import {
  ZoomableChart,
  DataPoint,
} from '@/components/molecules/charts/zoomable';

const getCreatedAfterFromTimeRange = (timeRange: string) => {
  const now = new Date();
  switch (timeRange) {
    case '1h':
      return new Date(now.getTime() - 60 * 60 * 1000).toISOString();
    case '6h':
      return new Date(now.getTime() - 6 * 60 * 60 * 1000).toISOString();
    case '1d':
      return new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
    case '7d':
      return new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString();
    default:
      return new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
  }
};

export const MetricsTab: FC = () => {
  const {
    data: managedWorker,
    metrics,
    setMetricsQuery,
  } = useManagedComputeDetail();

  const [defaultTimeRange, setDefaultTimeRange] = useState('1d');
  const [customTimeRange, setCustomTimeRange] = useState<
    string[] | undefined
  >();
  const [createdAfter, setCreatedAfter] = useState<string>(
    getCreatedAfterFromTimeRange('1d'),
  );
  const [finishedBefore, setFinishedBefore] = useState<string | undefined>();

  // Update metrics query when time range changes
  useEffect(() => {
    const newQuery = {
      after: createdAfter,
      before: finishedBefore,
    };
    setMetricsQuery(newQuery);
  }, [createdAfter, finishedBefore, setMetricsQuery]);

  // Auto-update time range
  useEffect(() => {
    const interval = setInterval(() => {
      if (customTimeRange) {
        return;
      }
      setCreatedAfter(getCreatedAfterFromTimeRange(defaultTimeRange));
    }, 60 * 1000);

    return () => clearInterval(interval);
  }, [defaultTimeRange, customTimeRange]);

  // Handle time range changes
  useEffect(() => {
    if (customTimeRange && customTimeRange.length === 2) {
      setCreatedAfter(customTimeRange[0]);
      setFinishedBefore(customTimeRange[1]);
    } else if (defaultTimeRange) {
      setCreatedAfter(getCreatedAfterFromTimeRange(defaultTimeRange));
      setFinishedBefore(undefined);
    }
  }, [defaultTimeRange, customTimeRange]);

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
        <div className="flex flex-row justify-end items-center gap-2">
          {customTimeRange
            ? [
                <Button
                  key="clear"
                  onClick={() => {
                    setCustomTimeRange(undefined);
                  }}
                  variant="outline"
                  size="sm"
                  className="text-xs h-9 py-2"
                >
                  <XCircleIcon className="h-[18px] w-[18px] mr-2" />
                  Clear
                </Button>,
                <DateTimePicker
                  key="after"
                  label="After"
                  date={createdAfter ? new Date(createdAfter) : undefined}
                  setDate={(date) => {
                    setCreatedAfter(date?.toISOString() || '');
                  }}
                />,
                <DateTimePicker
                  key="before"
                  label="Before"
                  date={finishedBefore ? new Date(finishedBefore) : undefined}
                  setDate={(date) => {
                    setFinishedBefore(date?.toISOString());
                  }}
                />,
              ]
            : null}
          <Select
            value={customTimeRange ? 'custom' : defaultTimeRange}
            onValueChange={(value) => {
              if (value !== 'custom') {
                setDefaultTimeRange(value);
                setCustomTimeRange(undefined);
              } else {
                setCustomTimeRange([
                  getCreatedAfterFromTimeRange(defaultTimeRange),
                  new Date().toISOString(),
                ]);
              }
            }}
          >
            <SelectTrigger className="w-fit">
              <SelectValue placeholder="Choose time range" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="1h">1 hour</SelectItem>
              <SelectItem value="6h">6 hours</SelectItem>
              <SelectItem value="1d">1 day</SelectItem>
              <SelectItem value="7d">7 days</SelectItem>
              <SelectItem value="custom">Custom</SelectItem>
            </SelectContent>
          </Select>
        </div>
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
        zoom={(createdAfter, createdBefore) => {
          setCustomTimeRange([createdAfter, createdBefore]);
        }}
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
        zoom={(createdAfter, createdBefore) => {
          setCustomTimeRange([createdAfter, createdBefore]);
        }}
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
        zoom={(createdAfter, createdBefore) => {
          setCustomTimeRange([createdAfter, createdBefore]);
        }}
        showYAxis={true}
      />
    </div>
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
