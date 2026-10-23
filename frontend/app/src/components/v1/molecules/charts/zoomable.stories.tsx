import { ZoomableChart, type DataPoint } from './zoomable';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { fn } from 'storybook/test';

type Series = 'succeeded' | 'failed' | 'cancelled';

const data: DataPoint<Series>[] = Array.from({ length: 48 }, (_, i) => ({
  date: new Date(Date.UTC(2026, 9, 20, 0, i * 30)).toISOString(),
  succeeded: Math.round(80 + 40 * Math.sin(i / 5)),
  failed: Math.round(6 + 5 * Math.cos(i / 3)),
  cancelled: i % 7 === 0 ? 4 : 1,
}));

const ZoomableRunsChart = ZoomableChart<Series>;

const meta = {
  title: 'Molecules/ZoomableChart',
  component: ZoomableRunsChart,
  args: {
    data,
    kind: 'bar',
    showYAxis: true,
    zoom: fn(),
    className: 'h-64',
  },
  argTypes: {
    kind: {
      control: 'inline-radio',
      options: ['bar', 'line', 'area'],
    },
    data: { control: false },
    colors: { control: 'object' },
  },
  decorators: [
    (Story) => (
      <div className="max-w-3xl">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof ZoomableRunsChart>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Bar: Story = {};

export const Line: Story = {
  args: {
    kind: 'line',
  },
};

export const Area: Story = {
  args: {
    kind: 'area',
  },
};
