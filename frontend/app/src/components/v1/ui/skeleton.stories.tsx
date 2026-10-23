import { Spinner } from './loading';
import { Skeleton } from './skeleton';
import type { Meta, StoryObj } from '@storybook/react-vite';

const meta = {
  title: 'UI/Skeleton',
  component: Skeleton,
  args: {
    className: 'h-4 w-64',
  },
} satisfies Meta<typeof Skeleton>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const TableRows: Story = {
  parameters: {
    controls: { disable: true },
  },
  render: () => (
    <div className="flex w-96 flex-col gap-3">
      {Array.from({ length: 5 }, (_, i) => (
        <div key={i} className="flex items-center gap-3">
          <Skeleton className="size-8 rounded-full" />
          <Skeleton className="h-4 flex-1" />
          <Skeleton className="h-4 w-16" />
        </div>
      ))}
    </div>
  ),
};

export const WithSpinner: Story = {
  parameters: {
    controls: { disable: true },
  },
  render: () => <Spinner />,
};
