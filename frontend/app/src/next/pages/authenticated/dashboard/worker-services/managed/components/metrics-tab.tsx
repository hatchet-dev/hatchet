import { Matrix } from '@/lib/api/generated/cloud/data-contracts';
import { useEffect, useMemo, useState } from 'react';
import { Separator } from '@/next/components/ui/separator';
import { DateTimePicker } from '@/components/molecules/time-picker/date-time-picker';
import { Button } from '@/next/components/ui/button';
import { XCircleIcon } from '@heroicons/react/24/outline';
import { DataPoint } from '@/components/v1/molecules/charts/zoomable';
import { useAtom } from 'jotai';
import { lastWorkerMetricsTimeRangeAtom } from '@/lib/atoms';
import { getCreatedAfterFromTimeRange } from '@/pages/main/workflow-runs/components/workflow-runs-table';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';

export function MetricsTab() {
  // const { metrics } = useManagedComputeDetail();

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

  // if (
  //   metrics.cpu?.isLoading ||
  //   metrics.memory?.isLoading ||
  //   metrics.disk?.isLoading
  // ) {
  //   return <Spinner />;
  // }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-row justify-between items-center mt-4">
        <h3 className="text-xl font-bold leading-tight text-foreground">
          Metrics
        </h3>
        <div className="flex flex-row justify-end items-center my-4 gap-2">
          {customTimeRange && [
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
      <h4 className="text-lg font-bold leading-tight text-foreground mt-2 ml-4">
        CPU
      </h4>
      <Separator />
      {/* {
        <ZoomableChart
          className="max-h-[25rem] min-h-[25rem]"
          data={transformToDataPoints(metrics.cpu.data)}
          kind="line"
          zoom={(createdAfter, createdBefore) => {
            setCustomTimeRange([createdAfter, createdBefore]);
          }}
          showYAxis={true}
        />
      } */}
      {/* <h4 className="text-lg font-bold leading-tight text-foreground mt-16 ml-4">
        Memory
      </h4>
      <Separator />
      {
        <ZoomableChart
          className="max-h-[25rem] min-h-[25rem]"
          data={transformToDataPoints(metrics.memory.data, (d) => {
            return d / (1000 * 1000);
          })}
          kind="line"
          zoom={(createdAfter, createdBefore) => {
            setCustomTimeRange([createdAfter, createdBefore]);
          }}
          showYAxis={true}
        />
      }
      <h4 className="text-lg font-bold leading-tight text-foreground mt-16 ml-4">
        Disk
      </h4>
      <Separator />
      {
        <ZoomableChart
          className="max-h-[25rem] min-h-[25rem]"
          data={transformToDataPoints(metrics.disk.data, (d) => {
            return d / (1000 * 1000);
          })}
          kind="line"
          zoom={(createdAfter, createdBefore) => {
            setCustomTimeRange([createdAfter, createdBefore]);
          }}
          showYAxis={true}
        />
      } */}
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
