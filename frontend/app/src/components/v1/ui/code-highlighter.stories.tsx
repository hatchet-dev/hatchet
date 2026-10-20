import { CodeHighlighter } from './code-highlighter';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { fn } from 'storybook/test';

const meta = {
  title: 'UI/CodeHighlighter',
  component: CodeHighlighter,
  args: {
    onCopy: fn(),
  },
  argTypes: {
    language: {
      control: 'select',
      options: ['typescript', 'json', 'yaml'],
    },
  },
  decorators: [
    (Story) => (
      <div className="max-w-2xl">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof CodeHighlighter>;

export default meta;
type Story = StoryObj<typeof meta>;

export const TypeScript: Story = {
  args: {
    language: 'typescript',
    code: `
import { hatchet } from '../hatchet-client';

export const simple = hatchet.task({
  name: 'simple',
  retries: 3,
  fn: async (input: { message: string }) => {
    return { transformed: input.message.toLowerCase() };
  },
});
`,
  },
};

export const Json: Story = {
  args: {
    language: 'json',
    code: JSON.stringify(
      { runId: 'b7c1e9d2', status: 'SUCCEEDED', output: { count: 42 } },
      null,
      2,
    ),
  },
};

export const Yaml: Story = {
  args: {
    language: 'yaml',
    code: `
name: on-push
on:
  events:
    - user:create
concurrency:
  maxRuns: 5
  limitStrategy: GROUP_ROUND_ROBIN
`,
  },
};

export const ScrollingWithoutCopy: Story = {
  args: {
    language: 'json',
    copy: false,
    maxHeight: '8rem',
    code: JSON.stringify(
      Array.from({ length: 20 }, (_, i) => ({ step: i, ok: true })),
      null,
      2,
    ),
  },
};
