"use client";

import {
  BarChart,
  Bar,
  CartesianGrid,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  ReferenceLine,
} from "recharts";
import { Zap } from "lucide-react";

import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

const chartData = [
  {
    connections: 10,
    throughput: 11004.0,
    latency: 0.907187,
  },
  {
    connections: 20,
    throughput: 16654.29,
    latency: 1.196766,
  },
  {
    connections: 30,
    throughput: 16506.04,
    latency: 1.809936,
  },
  {
    connections: 40,
    throughput: 16533.93,
    latency: 2.415203,
  },
  {
    connections: 50,
    throughput: 16797.25,
    latency: 2.971404,
  },
  {
    connections: 60,
    throughput: 16965.12,
    latency: 3.53187,
  },
  {
    connections: 70,
    throughput: 17006.43,
    latency: 4.110432,
  },
  {
    connections: 80,
    throughput: 16357.61,
    latency: 4.884029,
  },
  {
    connections: 90,
    throughput: 16875.42,
    latency: 5.326193,
  },
  {
    connections: 100,
    throughput: 17001.26,
    latency: 5.872899,
  },
];

// Calculate percentage increase from first to highest throughput
const maxThroughput = Math.max(...chartData.map((item) => item.throughput));

// Chart configuration
const chartConfig = {
  throughput: {
    color: "hsl(var(--primary))",
  },
};

export default function ConnectionsThroughputChart() {
  // Custom tooltip component
  const CustomTooltip = (props) => {
    const { active, payload } = props;
    if (active && payload && payload.length) {
      return (
        <div className="bg-background p-3 border border-border rounded-md shadow-md text-sm font-mono">
          <p className="font-semibold mb-2">{`Connections: ${payload[0]?.payload.connections}`}</p>
          <p className="text-primary flex items-center mb-1">
            <Zap className="h-3 w-3 mr-1" />
            Throughput: {payload[0]?.value?.toLocaleString()} rows/s
          </p>
          <p className="text-muted-foreground text-xs">
            Latency: {payload[0]?.payload.latency?.toFixed(2)} ms
          </p>
        </div>
      );
    }
    return null;
  };

  return (
    <Card className="w-full my-8">
      <CardHeader>
        <CardTitle>PostgreSQL Insert Performance</CardTitle>
        <CardDescription>
          Throughput vs Number of Concurrent Connections
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="h-96">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart
              data={chartData}
              margin={{
                left: 45,
                right: 20,
                top: 10,
                bottom: 20,
              }}
            >
              <CartesianGrid vertical={false} />
              <XAxis
                dataKey="connections"
                label={{
                  value: "Number of Connections",
                  position: "insideBottom",
                  offset: -10,
                  style: {
                    fontWeight: "bold",
                    fontSize: "11px",
                    textAnchor: "middle",
                  },
                }}
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                style={{ fontSize: "10px", userSelect: "none" }}
              />
              <YAxis
                label={{
                  value: "Throughput (rows/s)",
                  angle: -90,
                  position: "insideLeft",
                  offset: -30,
                  style: {
                    fontWeight: "bold",
                    fontSize: "11px",
                    textAnchor: "middle",
                  },
                }}
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                style={{ fontSize: "10px", userSelect: "none" }}
                domain={[0, "dataMax + 1000"]}
              />
              <Tooltip content={<CustomTooltip />} cursor={false} />
              <ReferenceLine
                y={chartData[0].throughput}
                stroke="gray"
                strokeDasharray="3 3"
                label={{
                  value: "Baseline",
                  position: "insideBottomLeft",
                  style: { fill: "gray", fontSize: 9 },
                }}
              />
              <Bar
                dataKey="throughput"
                name="Throughput"
                stroke={chartConfig.throughput.color}
                fill={chartConfig.throughput.color}
                isAnimationActive={false}
                fillOpacity={0.8}
              />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
      <CardFooter className="flex-col items-start gap-2 text-sm">
        <div className="leading-none text-muted-foreground mt-2">
          Tested with 100,000 rows
        </div>
      </CardFooter>
    </Card>
  );
}
