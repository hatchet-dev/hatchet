import React, { useMemo, useCallback } from 'react';
import { Group } from '@visx/group';
import { AreaClosed, Line, Bar } from '@visx/shape';
import {
  withTooltip,
  TooltipWithBounds,
  Tooltip,
  defaultStyles,
} from '@visx/tooltip';
import { GridRows, GridColumns } from '@visx/grid';
import { WithTooltipProvidedProps } from '@visx/tooltip/lib/enhancers/withTooltip';
import { scaleTime, scaleLinear } from '@visx/scale';
import { AxisLeft, AxisBottom } from '@visx/axis';
import { LinearGradient } from '@visx/gradient';
import { curveMonotoneX } from '@visx/curve';
import { localPoint } from '@visx/event';
import { max, extent, bisector } from '@visx/vendor/d3-array';
import { timeFormat } from '@visx/vendor/d3-time-format';
import { Text } from '@visx/text';

const getDate = (d: MetricValue) => d.date;
const getValue = (d: MetricValue) => d.value;

// format to 2 decimal places
export const format2Dec = (d: number) => {
  if (!d.toFixed) {
    return '0.00';
  }

  return `${d.toFixed(2)}`;
};

const bisectDate = bisector<MetricValue, Date>((d) => d.date).left;

export interface MetricValue {
  date: Date;
  value: number;
}
type TooltipData = MetricValue;

const formatDate = timeFormat('%y-%m-%d %I:%M:%S');

const accentColor = '#ffffff44';
const background = '#1E293B';
const background2 = '#8c77e0';
const accentColorDark = '#8c77e0';

const tooltipStyles = {
  ...defaultStyles,
  border: '1px solid white',
  color: 'white',
  background,
};

const axisColor = '#cecece';

const axisBottomTickLabelProps = {
  textAnchor: 'middle' as const,
  fontFamily: 'Arial',
  fontSize: 10,
  fill: axisColor,
};

const axisLeftTickLabelProps = {
  dx: '-0.25em',
  dy: '0.25em',
  fontFamily: 'Arial',
  fontSize: 10,
  textAnchor: 'end' as const,
  fill: axisColor,
};

export const formatPercentTooltip = (d: number) => `${format2Dec(d)}%`;

type AreaChartProps = {
  data: MetricValue[];
  kind: 'area' | 'bar';
  gradientColor?: string;
  width: number;
  height: number;
  hideBottomAxis?: boolean;
  hideLeftAxis?: boolean;
  children?: React.ReactNode;
  yLabel?: string;
  xLabel?: string;
  yDomain?: [number, number];
  xDomain?: [Date, Date];
  centerText?: string;
  tooltipFormat?: (d: number) => string;
};

export default withTooltip<AreaChartProps, TooltipData>(
  ({
    data,
    kind,
    gradientColor = background2,
    width,
    height,
    hideBottomAxis = false,
    hideLeftAxis = false,
    children,
    yLabel,
    xLabel,
    yDomain,
    xDomain,
    centerText,
    showTooltip,
    hideTooltip,
    tooltipFormat,
    tooltipData,
    tooltipTop = 0,
    tooltipLeft = 0,
  }: AreaChartProps & WithTooltipProvidedProps<TooltipData>) => {
    if (width < 10) {
      return null;
    }

    const innerWidth = width;
    const innerHeight = height;

    const dateScale = useMemo(
      () =>
        scaleTime<number>({
          range: [0, width],
          domain: xDomain || (extent(data, getDate) as [Date, Date]),
        }),
      [width, data, xDomain],
    );

    const yScale = useMemo(
      () =>
        scaleLinear<number>({
          range: [height, 0],
          domain: yDomain || [0, 1.3 * (max(data, getValue) || 0)],
          nice: true,
        }),
      [height, data, yDomain],
    );

    const handleTooltip = useCallback(
      (
        event:
          | React.TouchEvent<SVGRectElement>
          | React.MouseEvent<SVGRectElement>,
      ) => {
        const { x } = localPoint(event) || { x: 0 };
        const x0 = dateScale.invert(x);
        const index = bisectDate(data, x0, 1);
        const d0 = data[index - 1];
        const d1 = data[index];
        let d = d0;
        if (d1 && getDate(d1)) {
          d =
            x0.valueOf() - getDate(d0).valueOf() >
            getDate(d1).valueOf() - x0.valueOf()
              ? d1
              : d0;
        }

        showTooltip({
          tooltipData: d,
          tooltipLeft: x,
          tooltipTop: yScale(getValue(d)),
        });
      },
      [showTooltip, yScale, dateScale, data],
    );

    let barWidth = innerWidth / data.length;

    if (barWidth <= 5) {
      barWidth = 6;
    }

    return (
      <div>
        <svg width={width} height={height} overflow={'visible'}>
          {centerText && (
            <Text className="fill-foreground" x="50%" y="50%" dx={-200}>
              {centerText}
            </Text>
          )}
          <rect
            x={0}
            y={0}
            width={innerWidth}
            height={innerHeight}
            fill="url(#area-background-gradient)"
            rx={14}
          />
          <GridRows
            left={0}
            scale={yScale}
            width={innerWidth}
            height={innerHeight}
            strokeDasharray="1,3"
            stroke={accentColor}
            strokeOpacity={0.6}
            pointerEvents="none"
          />
          <GridColumns
            top={0}
            left={0}
            scale={dateScale}
            width={innerWidth}
            height={innerHeight}
            strokeDasharray="1,3"
            stroke={accentColor}
            strokeOpacity={0.6}
            pointerEvents="none"
          />
          <Group height={height} width={width}>
            <LinearGradient
              id="gradient"
              from={gradientColor}
              fromOpacity={1}
              to={gradientColor}
              toOpacity={0.2}
              height={innerHeight}
            />
            {kind == 'bar' &&
              data.map((d, i) => {
                if (i == 0) {
                  return (
                    <Bar
                      key={i}
                      x={dateScale(getDate(d)) || 0}
                      y={yScale(getValue(d)) || 0}
                      width={(barWidth - 4) / 2}
                      height={innerHeight - yScale(getValue(d)) || 0}
                      fill="url(#gradient)"
                      rx={2}
                    />
                  );
                }

                return (
                  <Bar
                    key={i}
                    x={(dateScale(getDate(d)) || 0) - barWidth / 2}
                    y={yScale(getValue(d)) || 0}
                    width={barWidth - 4}
                    height={innerHeight - yScale(getValue(d)) || 0}
                    fill="url(#gradient)"
                    rx={2}
                  />
                );
              })}
            {kind == 'area' && (
              <AreaClosed<MetricValue>
                data={data}
                x={(d) => dateScale(d.date) || 0}
                y={(d) => yScale(d.value) || 0}
                yScale={yScale}
                strokeWidth={1}
                stroke="url(#gradient)"
                fill="url(#gradient)"
                curve={curveMonotoneX}
                height={innerHeight}
              />
            )}
            {!hideBottomAxis && (
              <AxisBottom
                top={height}
                scale={dateScale}
                numTicks={width > 520 ? 10 : 5}
                stroke={axisColor}
                tickStroke={axisColor}
                tickLabelProps={axisBottomTickLabelProps}
                label={xLabel}
              />
            )}
            {!hideLeftAxis && (
              <AxisLeft
                scale={yScale}
                numTicks={5}
                stroke={axisColor}
                tickStroke={axisColor}
                tickLabelProps={axisLeftTickLabelProps}
                label={yLabel}
                labelClassName="text-white fill-foreground"
              />
            )}
            {children}
          </Group>
          <Bar
            x={0}
            y={0}
            width={innerWidth}
            height={innerHeight}
            fill="transparent"
            rx={14}
            onTouchStart={handleTooltip}
            onTouchMove={handleTooltip}
            onMouseMove={handleTooltip}
            onMouseLeave={() => hideTooltip()}
          />
          {data.length > 0 && tooltipData && (
            <g>
              <Line
                from={{ x: tooltipLeft, y: 0 }}
                to={{ x: tooltipLeft, y: innerHeight + 0 }}
                stroke={accentColorDark}
                strokeWidth={2}
                pointerEvents="none"
                strokeDasharray="5,2"
              />
              <circle
                cx={tooltipLeft}
                cy={tooltipTop + 1}
                r={4}
                fill="black"
                fillOpacity={0.1}
                stroke="black"
                strokeOpacity={0.1}
                strokeWidth={2}
                pointerEvents="none"
              />
              <circle
                cx={tooltipLeft}
                cy={tooltipTop}
                r={4}
                fill={accentColorDark}
                stroke="white"
                strokeWidth={2}
                pointerEvents="none"
              />
            </g>
          )}
        </svg>
        {data.length > 0 && tooltipData && (
          <div>
            <TooltipWithBounds
              key={Math.random()}
              top={tooltipTop - 24}
              left={tooltipLeft}
              style={tooltipStyles}
            >
              {tooltipFormat
                ? tooltipFormat(getValue(tooltipData))
                : getValue(tooltipData)}
            </TooltipWithBounds>
            <Tooltip
              top={innerHeight - 14}
              left={tooltipLeft}
              style={{
                ...defaultStyles,
                minWidth: 72,
                textAlign: 'center',
                transform: 'translateX(-50%)',
              }}
            >
              {formatDate(getDate(tooltipData))}
            </Tooltip>
          </div>
        )}
      </div>
    );
  },
);
