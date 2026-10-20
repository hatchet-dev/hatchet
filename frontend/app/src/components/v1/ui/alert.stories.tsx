import { Alert, AlertDescription, AlertTitle } from './alert';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { AlertTriangle, Info, XCircle } from 'lucide-react';

const meta = {
  title: 'UI/Alert',
  component: Alert,
  argTypes: {
    variant: {
      control: 'select',
      options: ['default', 'destructive', 'warn', 'info'],
    },
  },
  render: (args) => (
    <Alert {...args} className="max-w-lg">
      <AlertTitle>Heads up</AlertTitle>
      <AlertDescription>
        This workflow has no workers registered.
      </AlertDescription>
    </Alert>
  ),
} satisfies Meta<typeof Alert>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Variants: Story = {
  parameters: {
    controls: { disable: true },
  },
  render: () => (
    <div className="flex max-w-lg flex-col gap-3">
      <Alert variant="info">
        <Info className="size-4" />
        <AlertTitle>Info</AlertTitle>
        <AlertDescription>
          Runs older than 7 days are archived.
        </AlertDescription>
      </Alert>
      <Alert variant="warn">
        <AlertTriangle className="size-4" />
        <AlertTitle>Warning</AlertTitle>
        <AlertDescription>Queue depth is growing.</AlertDescription>
      </Alert>
      <Alert variant="destructive">
        <XCircle className="size-4" />
        <AlertTitle>Error</AlertTitle>
        <AlertDescription>The run failed after 3 retries.</AlertDescription>
      </Alert>
    </div>
  ),
};
