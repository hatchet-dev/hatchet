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

type RunStatus = 'SUCCEEDED' | 'FAILED';

// Mirrors a production tenant where each 30m bucket has ~50k successes and a
// few dozen failures: succeeded must fill the bar, failed stays a sliver.
const highVolumeRuns: DataPoint<RunStatus>[] = Array.from(
  { length: 48 },
  (_, i) => ({
    date: new Date(Date.UTC(2026, 10, 20, 0, i * 30)).toISOString(),
    SUCCEEDED: Math.round(48_000 + 4_000 * Math.sin(i / 4)),
    FAILED: i % 5 === 0 ? 0 : 20 + ((i * 7) % 20),
  }),
);

const ZoomableStatusChart = ZoomableChart<RunStatus>;

export const StackedHighVolume: StoryObj<typeof ZoomableStatusChart> = {
  render: (args) => <ZoomableStatusChart {...args} />,
  args: {
    data: highVolumeRuns,
    kind: 'bar',
    stacked: true,
    showYAxis: false,
    colors: {
      SUCCEEDED: 'rgb(34 197 94 / 0.5)',
      FAILED: 'hsl(var(--destructive))',
    },
    className: 'h-24 min-h-24',
  },
};

// One backfill-sized bucket squeezes every other bar to ~1px under a linear
// scale; compare with StackedHighVolumeWithSpikeSqrt.
export const StackedHighVolumeWithSpike: StoryObj<typeof ZoomableStatusChart> =
  {
    ...StackedHighVolume,
    args: {
      ...StackedHighVolume.args,
      data: highVolumeRuns.map((point, i) =>
        i === 40
          ? { ...point, SUCCEEDED: 2_500_000, FAILED: 120 }
          : i === 8
            ? { ...point, SUCCEEDED: 51_615, FAILED: 31 }
            : point,
      ),
    },
  };

export const StackedHighVolumeWithSpikeSqrt: StoryObj<
  typeof ZoomableStatusChart
> = {
  ...StackedHighVolumeWithSpike,
  args: {
    ...StackedHighVolumeWithSpike.args,
    yScale: 'sqrt',
  },
};
