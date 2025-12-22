import {
  DataPoint,
  ZoomableChart,
} from '@/components/v1/molecules/charts/zoomable';
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { Button } from '@/components/v1/ui/button';
import { Loading } from '@/components/v1/ui/loading';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { Separator } from '@/components/v1/ui/separator';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { queries } from '@/lib/api';
import {
  ManagedWorker,
  Matrix,
} from '@/lib/api/generated/cloud/data-contracts';
import { lastWorkerMetricsTimeRangeAtom } from '@/lib/atoms';
import { getCreatedAfterFromTimeRange } from '@/pages/main/workflow-runs/components/workflow-runs-table';
import { XCircleIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { useAtom } from 'jotai';
import { useEffect, useMemo, useState } from 'react';

export function ManagedWorkerMetrics({
  managedWorker,
}: {
  managedWorker: ManagedWorker;
}) {
  const { refetchInterval } = useRefetchInterval();
  const [defaultTimeRange, setDefaultTimeRange] = useAtom(
    lastWorkerMetricsTimeRangeAtom,
  );

  // customTimeRange does not get set in the atom,
  const [customTimeRange, setCustomTimeRange] = useState<
    string[] | undefined
  >();

  const [createdAfter, setCreatedAfter] = useState<string | undefined>(
    getCreatedAfterFromTimeRange(defaultTimeRange) ||
      new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
  );

  const [finishedBefore, setFinishedBefore] = useState<string | undefined>();

  // create a timer which updates the createdAfter date every minute
  useEffect(() => {
    const interval = setInterval(() => {
      if (customTimeRange) {
        return;
      }

      setCreatedAfter(
        getCreatedAfterFromTimeRange(defaultTimeRange) ||
          new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
      );
    }, 60 * 1000);

    return () => clearInterval(interval);
  }, [defaultTimeRange, customTimeRange]);

  // whenever the time range changes, update the createdAfter date
  useEffect(() => {
    if (customTimeRange && customTimeRange.length === 2) {
      setCreatedAfter(customTimeRange[0]);
      setFinishedBefore(customTimeRange[1]);
    } else if (defaultTimeRange) {
      setCreatedAfter(getCreatedAfterFromTimeRange(defaultTimeRange));
      setFinishedBefore(undefined);
    }
  }, [defaultTimeRange, customTimeRange, setCreatedAfter]);

  const queryParams = useMemo(() => {
    return {
      after: createdAfter,
      before: finishedBefore,
    };
  }, [createdAfter, finishedBefore]);

  const getCpuMetricsQuery = useQuery({
    ...queries.cloud.getManagedWorkerCpuMetrics(
      managedWorker?.metadata.id || '',
      queryParams,
    ),
    enabled: !!managedWorker,
    refetchInterval,
  });

  const getMemoryMetricsQuery = useQuery({
    ...queries.cloud.getManagedWorkerMemoryMetrics(
      managedWorker?.metadata.id || '',
      queryParams,
    ),
    enabled: !!managedWorker,
    refetchInterval,
  });

  const getDiskMetricsQuery = useQuery({
    ...queries.cloud.getManagedWorkerDiskMetrics(
      managedWorker?.metadata.id || '',
      queryParams,
    ),
    enabled: !!managedWorker,
    refetchInterval,
  });

  if (
    getCpuMetricsQuery.isLoading ||
    getMemoryMetricsQuery.isLoading ||
    getDiskMetricsQuery.isLoading ||
    !getCpuMetricsQuery.data ||
    !getMemoryMetricsQuery.data ||
    !getDiskMetricsQuery.data
  ) {
    return <Loading />;
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="mt-4 flex flex-row items-center justify-between">
        <h3 className="text-xl font-bold leading-tight text-foreground">
          Metrics
        </h3>
        <div className="my-4 flex flex-row items-center justify-end gap-2">
          {customTimeRange && [
            <Button
              key="clear"
              onClick={() => {
                setCustomTimeRange(undefined);
              }}
              variant="outline"
              size="sm"
              className="h-9 py-2 text-xs"
            >
              <XCircleIcon className="mr-2 h-[18px] w-[18px]" />
              Clear
            </Button>,
            <DateTimePicker
              key="after"
              label="After"
              date={createdAfter ? new Date(createdAfter) : undefined}
              setDate={(date) => {
                setCreatedAfter(date?.toISOString());
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
          ]}
          <Select
            value={customTimeRange ? 'custom' : defaultTimeRange}
            onValueChange={(value) => {
              if (value !== 'custom') {
                setDefaultTimeRange(value);
                setCustomTimeRange(undefined);
              } else {
                setCustomTimeRange([
                  getCreatedAfterFromTimeRange(value) ||
                    new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
                  new Date().toISOString(),
                ]);
              }
            }}
          >
            <SelectTrigger className="w-fit">
              <SelectValue id="timerange" placeholder="Choose time range" />
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
      <h4 className="ml-4 mt-2 text-lg font-bold leading-tight text-foreground">
        CPU
      </h4>
      <Separator />
      {
        <ZoomableChart
          className="max-h-[25rem] min-h-[25rem]"
          data={transformToDataPoints(getCpuMetricsQuery.data)}
          kind="line"
          zoom={(createdAfter, createdBefore) => {
            setCustomTimeRange([createdAfter, createdBefore]);
          }}
          showYAxis={true}
        />
      }
      <h4 className="ml-4 mt-16 text-lg font-bold leading-tight text-foreground">
        Memory
      </h4>
      <Separator />
      {
        <ZoomableChart
          className="max-h-[25rem] min-h-[25rem]"
          data={transformToDataPoints(getMemoryMetricsQuery.data, (d) => {
            return d / (1000 * 1000);
          })}
          kind="line"
          zoom={(createdAfter, createdBefore) => {
            setCustomTimeRange([createdAfter, createdBefore]);
          }}
          showYAxis={true}
        />
      }
      <h4 className="ml-4 mt-16 text-lg font-bold leading-tight text-foreground">
        Disk
      </h4>
      <Separator />
      {
        <ZoomableChart
          className="max-h-[25rem] min-h-[25rem]"
          data={transformToDataPoints(getDiskMetricsQuery.data, (d) => {
            return d / (1000 * 1000);
          })}
          kind="line"
          zoom={(createdAfter, createdBefore) => {
            setCustomTimeRange([createdAfter, createdBefore]);
          }}
          showYAxis={true}
        />
      }
    </div>
  );
}

function transformToDataPoints(
  matrix: Matrix,
  normalizer?: (n: number) => number,
): DataPoint<string>[] {
  const dataPointsMap: Record<string, DataPoint<string>> = {};

  matrix.forEach((sampleStream) => {
    // if we have instance or region, use that as the metricLabel
    let metricLabel = Object.values(sampleStream.metric || {}).join('-');

    if (sampleStream.metric?.instance && sampleStream.metric?.region) {
      metricLabel = `[${sampleStream.metric.region}] ${sampleStream.metric.instance}`;
    }

    (sampleStream.values || []).forEach(([timestamp, value]) => {
      const isoDate = new Date(timestamp * 1000).toISOString();

      if (!dataPointsMap[isoDate]) {
        const obj: any = {
          date: isoDate,
        };

        dataPointsMap[isoDate] = obj as DataPoint<string>;
      }

      let val = parseFloat(value);

      val = normalizer ? normalizer(val) : val;

      dataPointsMap[isoDate][metricLabel] = val;
    });
  });

  return Object.values(dataPointsMap);
}
