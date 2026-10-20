import { ToolbarType } from '../data-table/toolbar-type';
import { Combobox } from './combobox';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { fn } from 'storybook/test';

const statusOptions = [
  { label: 'Queued', value: 'QUEUED' },
  { label: 'Running', value: 'RUNNING' },
  { label: 'Succeeded', value: 'SUCCEEDED' },
  { label: 'Failed', value: 'FAILED' },
  { label: 'Cancelled', value: 'CANCELLED' },
];

function StatefulCombobox(args: React.ComponentProps<typeof Combobox>) {
  // Combobox mutates the array it's given, so never hand it Storybook's args.
  const [values, setValues] = useState(() => [...(args.values ?? [])]);

  return (
    <Combobox
      {...args}
      values={values}
      setValues={(next) => {
        args.setValues(next);
        setValues([...next]);
      }}
    />
  );
}

const meta = {
  title: 'Molecules/Combobox',
  component: Combobox,
  args: {
    title: 'Status',
    type: ToolbarType.Checkbox,
    values: [],
    setValues: fn(),
    options: statusOptions,
  },
  argTypes: {
    // The other ToolbarType values are handled by the data table, not here.
    type: {
      control: 'select',
      options: [
        ToolbarType.Checkbox,
        ToolbarType.Radio,
        ToolbarType.KeyValue,
        ToolbarType.Array,
      ],
    },
    icon: { control: false },
    // Setting these turns off the built-in filtering of options.
    searchValue: { control: false },
    onSearchChange: { control: false },
  },
  render: (args) => (
    <StatefulCombobox
      key={`${args.type}:${JSON.stringify(args.values)}`}
      {...args}
    />
  ),
} satisfies Meta<typeof Combobox>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Checkbox: Story = {};

export const Radio: Story = {
  args: {
    type: ToolbarType.Radio,
    values: ['RUNNING'],
  },
};

export const KeyValue: Story = {
  args: {
    title: 'Metadata',
    type: ToolbarType.KeyValue,
    options: undefined,
    values: ['env:production'],
  },
};

export const ArrayInput: Story = {
  args: {
    title: 'Worker IDs',
    type: ToolbarType.Array,
    options: undefined,
  },
};
