import { Button } from './button';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from './card';
import type { Meta, StoryObj } from '@storybook/react-vite';

const meta = {
  title: 'UI/Card',
  component: Card,
  argTypes: {
    variant: {
      control: 'select',
      options: ['default', 'light'],
    },
  },
  render: (args) => (
    <Card {...args} className="w-96">
      <CardHeader>
        <CardTitle>Worker pool</CardTitle>
        <CardDescription>3 workers connected in this tenant.</CardDescription>
      </CardHeader>
      <CardContent className="text-sm">
        Slots are shared across all workflows registered on these workers.
      </CardContent>
      <CardFooter>
        <Button size="sm">View workers</Button>
      </CardFooter>
    </Card>
  ),
} satisfies Meta<typeof Card>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Light: Story = {
  args: {
    variant: 'light',
  },
};
