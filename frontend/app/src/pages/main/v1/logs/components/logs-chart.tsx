import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from '@/components/v1/ui/chart';
import { V1LogsPointMetric } from '@/lib/api';
import { useState } from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  CartesianGrid,
  ReferenceArea,
  ResponsiveContainer,
} from 'recharts';

const CHART_CONFIG = {
  DEBUG: { label: 'Debug', color: 'rgba(107, 114, 128, 0.55)' },
  INFO: { label: 'Info', color: 'rgba(34, 197, 94, 0.65)' },
  WARN: { label: 'Warn', color: 'rgba(234, 179, 8, 0.8)' },
  ERROR: { label: 'Error', color: 'rgba(239, 68, 68, 0.9)' },
} satisfies ChartConfig;

function getNextPointTime(time: string, points: V1LogsPointMetric[]): string {
  const idx = points.findIndex((p) => p.time === time);
  if (idx === -1) return time;
  if (idx === points.length - 1) {
    const last = new Date(points[idx].time).getTime();
    const prev = new Date(points[idx - 1].time).getTime();
    return new Date(last + (last - prev)).toISOString();
  }
  return points[idx + 1].time;
}

function getPrevPointTime(time: string, points: V1LogsPointMetric[]): string {
  const idx = points.findIndex((p) => p.time === time);
  if (idx === -1) return time;
  if (idx === 0) {
    const first = new Date(points[0].time).getTime();
    const second = new Date(points[1].time).getTime();
    return new Date(first - (second - first)).toISOString();
  }
  return points[idx - 1].time;
}

function formatXAxis(tickItem: string, minDate: Date, maxDate: Date): string {
  const date = new Date(tickItem);
  const timeDiff = maxDate.getTime() - minDate.getTime();
  const oneDay = 24 * 60 * 60 * 1000;
  const sevenDays = 7 * oneDay;

  if (timeDiff > sevenDays) {
    return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
  } else if (timeDiff > oneDay) {
    return `${date.toLocaleDateString([], { month: 'short', day: 'numeric' })} ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
  } else {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
}

interface LogsChartProps {
  metrics: V1LogsPointMetric[];
  since: string;
  until?: string;
  onZoom: (since: string, until: string) => void;
}

export function LogsChart({ metrics, since, until, onZoom }: LogsChartProps) {
  const [refAreaLeft, setRefAreaLeft] = useState<string | null>(null);
  const [refAreaRight, setRefAreaRight] = useState<string | null>(null);
  const [actualLeft, setActualLeft] = useState<string | null>(null);
  const [actualRight, setActualRight] = useState<string | null>(null);
  const [isSelecting, setIsSelecting] = useState(false);

  const minDate = new Date(since);
  const maxDate = new Date(until ?? new Date().toISOString());

  const handleMouseDown = (e: any) => {
    if (e?.activeLabel) {
      setRefAreaLeft(e.activeLabel);
      setActualLeft(getPrevPointTime(e.activeLabel, metrics));
      setIsSelecting(true);
    }
  };

  const handleMouseMove = (e: any) => {
    if (isSelecting && e?.activeLabel) {
      setRefAreaRight(e.activeLabel);
      setActualRight(getNextPointTime(e.activeLabel, metrics));
    }
  };

  const handleMouseUp = () => {
    if (actualLeft && actualRight) {
      const [left, right] = [actualLeft, actualRight].sort();
      onZoom(left, right);
    }
    setRefAreaLeft(null);
    setRefAreaRight(null);
    setActualLeft(null);
    setActualRight(null);
    setIsSelecting(false);
  };

  return (
    <ChartContainer config={CHART_CONFIG} className="h-24 min-h-24 w-full">
      <div className="h-full" style={{ touchAction: 'none' }}>
        <ResponsiveContainer width="100%" height="100%">
          <BarChart
            data={metrics}
            margin={{ left: 0, right: 0, top: 0, bottom: 0 }}
            onMouseDown={handleMouseDown}
            onMouseMove={handleMouseMove}
            onMouseUp={handleMouseUp}
          >
            <CartesianGrid vertical={false} />
            <XAxis
              dataKey="time"
              tickFormatter={(v) => formatXAxis(v, minDate, maxDate)}
              tickLine={false}
              axisLine={false}
              tickMargin={4}
              minTickGap={16}
              style={{ fontSize: '10px', userSelect: 'none' }}
            />
            <ChartTooltip
              content={
                <ChartTooltipContent
                  className="w-[150px] font-mono text-xs"
                  labelFormatter={(v) => new Date(v).toLocaleString()}
                />
              }
            />
            <Bar dataKey="DEBUG" stackId="normal" fill={CHART_CONFIG.DEBUG.color} isAnimationActive={false} />
            <Bar dataKey="INFO" stackId="normal" fill={CHART_CONFIG.INFO.color} isAnimationActive={false} />
            <Bar dataKey="WARN" stackId="normal" fill={CHART_CONFIG.WARN.color} isAnimationActive={false} />
            <Bar dataKey="ERROR" stackId="error" fill={CHART_CONFIG.ERROR.color} isAnimationActive={false} />

            {refAreaLeft && refAreaRight && (
              <ReferenceArea
                x1={refAreaLeft}
                x2={refAreaRight}
                strokeOpacity={0.3}
                fill="hsl(var(--foreground))"
                fillOpacity={0.1}
              />
            )}
          </BarChart>
        </ResponsiveContainer>
      </div>
    </ChartContainer>
  );
}
