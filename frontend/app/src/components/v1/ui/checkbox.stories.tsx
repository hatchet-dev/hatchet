import { Checkbox } from './checkbox';
import { Label } from './label';
import { Switch } from './switch';
import type { Meta, StoryObj } from '@storybook/react-vite';

const meta = {
  title: 'UI/Checkbox',
  component: Checkbox,
  args: {
    defaultChecked: true,
  },
  argTypes: {
    defaultChecked: { control: 'boolean' },
    disabled: { control: 'boolean' },
  },
  render: (args) => <Checkbox key={String(args.defaultChecked)} {...args} />,
} satisfies Meta<typeof Checkbox>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithLabel: Story = {
  render: (args) => (
    <div className="flex items-center gap-2">
      <Checkbox key={String(args.defaultChecked)} {...args} id="retry" />
      <Label htmlFor="retry">Retry on failure</Label>
    </div>
  ),
};

export const Disabled: Story = {
  args: {
    disabled: true,
  },
};

export const SwitchControl: Story = {
  name: 'Switch',
  parameters: {
    controls: { disable: true },
  },
  render: () => (
    <div className="flex items-center gap-2">
      <Switch id="alerts" defaultChecked />
      <Label htmlFor="alerts">Email alerts</Label>
    </div>
  ),
};
