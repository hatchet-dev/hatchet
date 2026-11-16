import JsonSchemaForm from './json-schema-form';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { fn } from 'storybook/test';

const meta = {
  title: 'Molecules/JsonSchemaForm',
  component: JsonSchemaForm,
  args: {
    initialValue: {},
    onChange: fn(),
  },
  render: (args) => (
    <JsonSchemaForm
      key={JSON.stringify([args.schema, args.initialValue])}
      {...args}
    />
  ),
  decorators: [
    (Story) => (
      <div className="max-w-xl">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof JsonSchemaForm>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Primitives: Story = {
  args: {
    schema: {
      type: 'object',
      required: ['email'],
      properties: {
        email: { type: 'string', format: 'email', title: 'Email' },
        name: { type: 'string', title: 'Name' },
        retries: { type: 'integer', minimum: 0, maximum: 10, default: 3 },
        dryRun: { type: 'boolean', title: 'Dry run', default: false },
      },
    },
  },
};

export const EnumsAndDates: Story = {
  args: {
    schema: {
      type: 'object',
      properties: {
        priority: {
          type: 'string',
          enum: ['low', 'medium', 'high'],
          default: 'medium',
        },
        scheduledAt: { type: 'string', format: 'date-time' },
      },
    },
  },
};

export const NestedAndArrays: Story = {
  args: {
    schema: {
      type: 'object',
      properties: {
        customer: {
          type: 'object',
          title: 'Customer',
          properties: {
            id: { type: 'string' },
            tier: { type: 'string', enum: ['free', 'pro', 'enterprise'] },
          },
        },
        items: {
          type: 'array',
          title: 'Items',
          items: {
            type: 'object',
            properties: {
              sku: { type: 'string' },
              quantity: { type: 'integer', minimum: 1, default: 1 },
            },
          },
        },
      },
    },
    initialValue: {
      customer: { id: 'cus_123', tier: 'pro' },
      items: [{ sku: 'SKU-1', quantity: 2 }],
    },
  },
};
