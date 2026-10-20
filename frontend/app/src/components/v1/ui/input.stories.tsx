import { Input } from './input';
import { Label } from './label';
import { Textarea } from './textarea';
import type { Meta, StoryObj } from '@storybook/react-vite';

const meta = {
  title: 'UI/Input',
  component: Input,
  args: {
    type: 'text',
    placeholder: 'my-workflow',
  },
  argTypes: {
    type: {
      control: 'select',
      options: ['text', 'password', 'email', 'number', 'search'],
    },
  },
  decorators: [
    (Story) => (
      <div className="w-80">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof Input>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithLabel: Story = {
  render: (args) => (
    <div className="grid gap-2">
      <Label htmlFor="workflow-name">Workflow name</Label>
      <Input {...args} id="workflow-name" />
    </div>
  ),
};

export const Disabled: Story = {
  args: {
    disabled: true,
    defaultValue: 'read-only',
  },
};

export const Multiline: Story = {
  parameters: {
    controls: { disable: true },
  },
  render: () => <Textarea placeholder='{ "input": "value" }' rows={4} />,
};
