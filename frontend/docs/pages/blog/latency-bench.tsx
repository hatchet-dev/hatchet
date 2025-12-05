"use client";

import { useState } from "react";
import {
  BarChart,
  LineChart,
  Line,
  Bar,
  CartesianGrid,
  XAxis,
  YAxis,
  Legend,
  Tooltip,
  ResponsiveContainer,
  ReferenceArea,
} from "recharts";
import { TrendingUp, TrendingDown, Clock, Zap } from "lucide-react";

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
    batchSize: 5,
    flushInterval: 0.5,
    label: "5/0.5ms",
    throughput: 33372,
    latency: 5.78,
  },
  {
    batchSize: 10,
    flushInterval: 1,
    label: "10/1ms",
    throughput: 69896,
    latency: 5.28,
  },
  {
    batchSize: 25,
    flushInterval: 2.5,
    label: "25/2.5ms",
    throughput: 89071,
    latency: 9.6,
  },
  {
    batchSize: 50,
    flushInterval: 5,
    label: "50/5ms",
    throughput: 91680,
    latency: 14.91,
  },
  {
    batchSize: 100,
    flushInterval: 10,
    label: "100/10ms",
    throughput: 98633,
    latency: 14.01,
  },
  {
    batchSize: 200,
    flushInterval: 20,
    label: "200/20ms",
    throughput: 107647,
    latency: 17.89,
  },
];

// Calculate percentage increases for the footer
const throughputIncrease = (
  ((chartData[chartData.length - 1].throughput - chartData[0].throughput) /
    chartData[0].throughput) *
  100
).toFixed(1);
const latencyIncrease = (
  ((chartData[chartData.length - 1].latency - chartData[0].latency) /
    chartData[0].latency) *
  100
).toFixed(1);

// Chart configuration
const chartConfig = {
  throughput: {
    label: "Throughput",
    color: "hsl(var(--chart-1))",
  },
  latency: {
    label: "Latency",
    color: "hsl(var(--primary))",
  },
};

export default function PerformanceChart() {
  const [activeTab, setActiveTab] = useState("both");
  const [refAreaLeft, setRefAreaLeft] = useState(null);
  const [refAreaRight, setRefAreaRight] = useState(null);

  // Custom tooltip component to show both metrics
  const CustomTooltip = (props) => {
    const { active, payload } = props;
    if (active && payload && payload.length) {
      return (
        <div className="bg-background p-3 border border-border rounded-md shadow-md text-sm font-mono">
          <p className="font-semibold mb-2">{`Batch: ${payload[0]?.payload.batchSize}, Flush: ${payload[0]?.payload.flushInterval}ms`}</p>
          {activeTab === "both" || activeTab === "throughput" ? (
            <p className="text-primary flex items-center mb-1">
              <Zap className="h-3 w-3 mr-1" />
              Throughput: {payload[0]?.value?.toLocaleString()} rows/s
            </p>
          ) : null}
          {activeTab === "both" || activeTab === "latency" ? (
            <p className="text-primary flex items-center">
              <Clock className="h-3 w-3 mr-1" />
              Latency:{" "}
              {(activeTab === "both"
                ? payload[1]?.value
                : payload[0]?.value
              )?.toFixed(2)}{" "}
              ms
            </p>
          ) : null}
        </div>
      );
    }
    return null;
  };

  // Chart interaction handlers (for zooming functionality)
  const handleMouseDown = (e) => {
    if (e && e.activeLabel) {
      setRefAreaLeft(e.activeLabel);
    }
  };

  const handleMouseMove = (e) => {
    if (refAreaLeft && e && e.activeLabel) {
      setRefAreaRight(e.activeLabel);
    }
  };

  const handleMouseUp = () => {
    // Reset reference area when mouse is released
    setRefAreaLeft(null);
    setRefAreaRight(null);
  };

  // Common label and axis style with foreground color
  const labelStyle = {
    fontWeight: "bold",
    fontSize: "11px",
    textAnchor: "middle",
    fill: "hsl(var(--foreground))",
  };

  // Style for axis text (without userSelect property)
  const axisStyle = {
    fontSize: "10px",
    fill: "hsl(var(--foreground))",
  };

  return (
    <Card className="w-full my-8">
      <CardHeader className="gap-2">
        <CardTitle>PostgreSQL COPY Performance with Buffered Writes</CardTitle>
        <CardDescription>
          Throughput and Latency vs Batch Size/Flush Interval
        </CardDescription>
        <div className="flex space-x-2 mt-2">
          <button
            className={`px-3 py-1 text-sm rounded-md ${activeTab === "both" ? "bg-primary/10 text-primary font-medium" : "bg-muted"}`}
            onClick={() => setActiveTab("both")}
          >
            Both
          </button>
          <button
            className={`px-3 py-1 text-sm rounded-md ${activeTab === "throughput" ? "bg-primary/10 text-primary font-medium" : "bg-muted"}`}
            onClick={() => setActiveTab("throughput")}
          >
            Throughput
          </button>
          <button
            className={`px-3 py-1 text-sm rounded-md ${activeTab === "latency" ? "bg-primary/10 text-primary font-medium" : "bg-muted"}`}
            onClick={() => setActiveTab("latency")}
          >
            Latency
          </button>
        </div>
      </CardHeader>
      <CardContent>
        <div className="h-96">
          <ResponsiveContainer width="100%" height="100%">
            {activeTab === "both" ? (
              <ComposedChart
                data={chartData}
                margin={{
                  left: 35,
                  right: 35,
                  top: 10,
                  bottom: 20,
                }}
                onMouseDown={handleMouseDown}
                onMouseMove={handleMouseMove}
                onMouseUp={handleMouseUp}
              >
                <CartesianGrid vertical={false} />
                <XAxis
                  dataKey="label"
                  label={{
                    value: "Batch Size/Flush Interval",
                    position: "insideBottom",
                    offset: -10,
                    style: labelStyle,
                  }}
                  tickLine={false}
                  axisLine={false}
                  tickMargin={8}
                  minTickGap={16}
                  style={axisStyle}
                />
                <YAxis
                  yAxisId="left"
                  label={{
                    value: "Throughput (rows/s)",
                    angle: -90,
                    position: "insideLeft",
                    offset: -30,
                    style: labelStyle,
                  }}
                  orientation="left"
                  domain={[0, "auto"]}
                  tickLine={false}
                  axisLine={false}
                  tickMargin={8}
                  style={axisStyle}
                />
                <YAxis
                  yAxisId="right"
                  label={{
                    value: "Latency (ms)",
                    angle: 90,
                    position: "insideRight",
                    offset: -25,
                    style: labelStyle,
                  }}
                  orientation="right"
                  domain={[0, "auto"]}
                  tickLine={false}
                  axisLine={false}
                  tickMargin={8}
                  style={axisStyle}
                />
                <Tooltip content={<CustomTooltip />} cursor={false} />
                <Bar
                  yAxisId="left"
                  dataKey="throughput"
                  name="Throughput"
                  stroke={chartConfig.throughput.color}
                  fill={chartConfig.throughput.color}
                  isAnimationActive={false}
                  fillOpacity={1}
                />
                <Line
                  yAxisId="right"
                  type="monotone"
                  dataKey="latency"
                  name="Latency"
                  stroke={chartConfig.latency.color}
                  strokeWidth={2}
                  dot={{ r: 3, fill: chartConfig.latency.color }}
                  isAnimationActive={false}
                />

                {refAreaLeft && refAreaRight && (
                  <ReferenceArea
                    x1={refAreaLeft}
                    x2={refAreaRight}
                    strokeOpacity={0.3}
                    fill="hsl(var(--foreground))"
                    fillOpacity={0.1}
                  />
                )}
              </ComposedChart>
            ) : activeTab === "throughput" ? (
              <BarChart
                data={chartData}
                margin={{
                  left: 45,
                  right: 20,
                  top: 10,
                  bottom: 20,
                }}
                onMouseDown={handleMouseDown}
                onMouseMove={handleMouseMove}
                onMouseUp={handleMouseUp}
              >
                <CartesianGrid vertical={false} />
                <XAxis
                  dataKey="label"
                  label={{
                    value: "Batch Size/Flush Interval",
                    position: "insideBottom",
                    offset: -10,
                    style: labelStyle,
                  }}
                  tickLine={false}
                  axisLine={false}
                  tickMargin={8}
                  minTickGap={16}
                  style={axisStyle}
                />
                <YAxis
                  label={{
                    value: "Throughput (rows/s)",
                    angle: -90,
                    position: "insideLeft",
                    offset: -30,
                    style: labelStyle,
                  }}
                  tickLine={false}
                  axisLine={false}
                  tickMargin={8}
                  style={axisStyle}
                />
                <Tooltip content={<CustomTooltip />} cursor={false} />
                <Bar
                  dataKey="throughput"
                  name="Throughput"
                  stroke={chartConfig.throughput.color}
                  fill={chartConfig.throughput.color}
                  isAnimationActive={false}
                  fillOpacity={1}
                />

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
            ) : (
              <LineChart
                data={chartData}
                margin={{
                  left: 45,
                  right: 20,
                  top: 10,
                  bottom: 20,
                }}
                onMouseDown={handleMouseDown}
                onMouseMove={handleMouseMove}
                onMouseUp={handleMouseUp}
              >
                <CartesianGrid vertical={false} />
                <XAxis
                  dataKey="label"
                  label={{
                    value: "Batch Size/Flush Interval",
                    position: "insideBottom",
                    offset: -10,
                    style: labelStyle,
                  }}
                  tickLine={false}
                  axisLine={false}
                  tickMargin={8}
                  minTickGap={16}
                  style={axisStyle}
                />
                <YAxis
                  label={{
                    value: "Latency (ms)",
                    angle: -90,
                    position: "insideLeft",
                    offset: -30,
                    style: labelStyle,
                  }}
                  tickLine={false}
                  axisLine={false}
                  tickMargin={8}
                  style={axisStyle}
                />
                <Tooltip content={<CustomTooltip />} cursor={false} />
                <Line
                  type="monotone"
                  dataKey="latency"
                  name="Latency"
                  stroke={chartConfig.latency.color}
                  strokeWidth={2}
                  dot={{ r: 3, fill: chartConfig.latency.color }}
                  isAnimationActive={false}
                />

                {refAreaLeft && refAreaRight && (
                  <ReferenceArea
                    x1={refAreaLeft}
                    x2={refAreaRight}
                    strokeOpacity={0.3}
                    fill="hsl(var(--foreground))"
                    fillOpacity={0.1}
                  />
                )}
              </LineChart>
            )}
          </ResponsiveContainer>
        </div>
      </CardContent>
      <CardFooter className="flex-col items-start gap-2 text-sm">
        <div className="leading-none text-muted-foreground mt-2">
          Testing with 20 connections over 30 seconds
        </div>
      </CardFooter>
    </Card>
  );
}

// Import ComposedChart from recharts
import { ComposedChart } from "recharts";
