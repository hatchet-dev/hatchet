import { Badge } from './badge';
import type { Meta, StoryObj } from '@storybook/react-vite';

const meta = {
  title: 'UI/Badge',
  component: Badge,
  args: {
    children: 'Badge',
  },
  argTypes: {
    variant: {
      control: 'select',
      options: [
        'default',
        'secondary',
        'destructive',
        'outline',
        'outlineDestructive',
        'successful',
        'failed',
        'inProgress',
        'queued',
        'cancelled',
        'evicted',
      ],
    },
  },
} satisfies Meta<typeof Badge>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const RunStatuses: Story = {
  argTypes: {
    variant: { control: false },
  },
  render: (args) => (
    <div className="flex flex-wrap items-center gap-2">
      <Badge {...args} variant="queued">
        Queued
      </Badge>
      <Badge {...args} variant="inProgress">
        Running
      </Badge>
      <Badge {...args} variant="successful">
        Succeeded
      </Badge>
      <Badge {...args} variant="failed">
        Failed
      </Badge>
      <Badge {...args} variant="cancelled">
        Cancelled
      </Badge>
      <Badge {...args} variant="evicted">
        Evicted
      </Badge>
    </div>
  ),
};
