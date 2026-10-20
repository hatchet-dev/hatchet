import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './select';
import type { Meta, StoryObj } from '@storybook/react-vite';

const meta = {
  title: 'UI/Select',
  component: Select,
  args: {
    defaultValue: '1h',
  },
  argTypes: {
    defaultValue: {
      control: 'select',
      options: ['15m', '1h', '24h', '7d'],
    },
  },
  render: (args) => (
    <Select key={args.defaultValue} {...args}>
      <SelectTrigger className="w-48">
        <SelectValue placeholder="Time range" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="15m">Last 15 minutes</SelectItem>
        <SelectItem value="1h">Last hour</SelectItem>
        <SelectItem value="24h">Last 24 hours</SelectItem>
        <SelectItem value="7d">Last 7 days</SelectItem>
      </SelectContent>
    </Select>
  ),
} satisfies Meta<typeof Select>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Placeholder: Story = {
  args: {
    defaultValue: undefined,
  },
};

export const Disabled: Story = {
  args: {
    disabled: true,
  },
};
