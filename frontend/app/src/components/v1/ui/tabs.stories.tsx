import { Tabs, TabsContent, TabsList, TabsTrigger } from './tabs';
import type { Meta, StoryObj } from '@storybook/react-vite';

const meta = {
  title: 'UI/Tabs',
  component: Tabs,
  args: {
    defaultValue: 'runs',
  },
  argTypes: {
    defaultValue: {
      control: 'inline-radio',
      options: ['runs', 'workers'],
    },
  },
} satisfies Meta<typeof Tabs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (args) => (
    <Tabs key={args.defaultValue} {...args}>
      <TabsList>
        <TabsTrigger value="runs">Runs</TabsTrigger>
        <TabsTrigger value="workers">Workers</TabsTrigger>
        <TabsTrigger value="settings" disabled>
          Settings
        </TabsTrigger>
      </TabsList>
      <TabsContent value="runs" className="pt-4 text-sm">
        Recent runs.
      </TabsContent>
      <TabsContent value="workers" className="pt-4 text-sm">
        Connected workers.
      </TabsContent>
    </Tabs>
  ),
};

export const Underlined: Story = {
  render: (args) => (
    <Tabs key={args.defaultValue} {...args}>
      <TabsList layout="underlined">
        <TabsTrigger variant="underlined" value="runs">
          Runs
        </TabsTrigger>
        <TabsTrigger variant="underlined" value="workers">
          Workers
        </TabsTrigger>
      </TabsList>
      <TabsContent value="runs" className="pt-4 text-sm">
        Recent runs.
      </TabsContent>
      <TabsContent value="workers" className="pt-4 text-sm">
        Connected workers.
      </TabsContent>
    </Tabs>
  ),
};
