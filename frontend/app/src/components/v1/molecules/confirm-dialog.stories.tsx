import { ConfirmDialog } from './confirm-dialog';
import { Button } from '@/components/v1/ui/button';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { fn } from 'storybook/test';

const meta = {
  title: 'Molecules/ConfirmDialog',
  component: ConfirmDialog,
  args: {
    title: 'Cancel run',
    description:
      'Are you sure you want to cancel this run? Running tasks will be stopped.',
    submitLabel: 'Cancel run',
    cancelLabel: 'Keep running',
    isLoading: false,
    isOpen: true,
    onSubmit: fn(),
    onCancel: fn(),
  },
  argTypes: {
    submitVariant: {
      control: 'select',
      options: ['destructive', 'default', 'outline', 'secondary'],
    },
    // Accepted by ConfirmDialog but not applied to anything.
    className: { control: false },
  },
} satisfies Meta<typeof ConfirmDialog>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Loading: Story = {
  args: {
    isLoading: true,
  },
};

export const NonDestructive: Story = {
  args: {
    title: 'Replay runs',
    description: 'Replay the 12 selected runs with their original input?',
    submitLabel: 'Replay',
    cancelLabel: 'Cancel',
    submitVariant: 'default',
  },
};

export const Interactive: Story = {
  argTypes: {
    isOpen: { control: false },
    isLoading: { control: false },
  },
  render: function Render(args) {
    const [isOpen, setIsOpen] = useState(false);
    const [isLoading, setIsLoading] = useState(false);

    return (
      <>
        <Button variant="destructive" onClick={() => setIsOpen(true)}>
          Cancel run
        </Button>
        <ConfirmDialog
          {...args}
          isOpen={isOpen}
          isLoading={isLoading}
          onCancel={() => {
            args.onCancel();
            setIsOpen(false);
          }}
          onSubmit={() => {
            args.onSubmit();
            setIsLoading(true);
            setTimeout(() => {
              setIsLoading(false);
              setIsOpen(false);
            }, 1000);
          }}
        />
      </>
    );
  },
};
