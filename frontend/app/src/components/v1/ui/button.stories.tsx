import { Button } from './button';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Plus } from 'lucide-react';

const meta = {
  title: 'UI/Button',
  component: Button,
  args: {
    children: 'Button',
  },
  argTypes: {
    variant: {
      control: 'select',
      options: [
        'default',
        'destructive',
        'outline',
        'secondary',
        'ghost',
        'link',
        'icon',
        'cta',
      ],
    },
    size: {
      control: 'select',
      options: ['default', 'sm', 'lg', 'icon', 'xs'],
    },
    // Slot needs a single element child, which a text control can't provide.
    asChild: { control: false },
    leftIcon: { control: false },
    rightIcon: { control: false },
  },
} satisfies Meta<typeof Button>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Variants: Story = {
  argTypes: {
    variant: { control: false },
  },
  render: (args) => (
    <div className="flex flex-wrap items-center gap-2">
      <Button {...args} variant="default">
        Default
      </Button>
      <Button {...args} variant="secondary">
        Secondary
      </Button>
      <Button {...args} variant="outline">
        Outline
      </Button>
      <Button {...args} variant="ghost">
        Ghost
      </Button>
      <Button {...args} variant="destructive">
        Destructive
      </Button>
      <Button {...args} variant="link">
        Link
      </Button>
      <Button {...args} variant="cta">
        CTA
      </Button>
    </div>
  ),
};

export const WithIcon: Story = {
  args: {
    leftIcon: <Plus className="size-4" />,
    children: 'Add worker',
  },
};

export const WithHoverText: Story = {
  args: {
    hoverText: 'Tooltip on hover',
  },
};

export const Disabled: Story = {
  args: {
    disabled: true,
  },
};
