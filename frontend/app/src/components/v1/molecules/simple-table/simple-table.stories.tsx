import { SimpleTable } from './simple-table';
import { Badge, type BadgeProps } from '@/components/v1/ui/badge';
import type { Meta, StoryObj } from '@storybook/react-vite';

type Run = {
  id: string;
  workflow: string;
  status: 'QUEUED' | 'RUNNING' | 'SUCCEEDED' | 'FAILED' | 'CANCELLED';
  duration: string;
};

const statusVariant: Record<Run['status'], BadgeProps['variant']> = {
  QUEUED: 'queued',
  RUNNING: 'inProgress',
  SUCCEEDED: 'successful',
  FAILED: 'failed',
  CANCELLED: 'cancelled',
};

const runs: Run[] = [
  {
    id: '3f2a9c1e',
    workflow: 'process-order',
    status: 'SUCCEEDED',
    duration: '1.2s',
  },
  { id: '8b71d0aa', workflow: 'send-email', status: 'RUNNING', duration: '—' },
  {
    id: 'c04e55f9',
    workflow: 'process-order',
    status: 'FAILED',
    duration: '32.7s',
  },
  {
    id: '19ad6e72',
    workflow: 'generate-report',
    status: 'QUEUED',
    duration: '—',
  },
  {
    id: 'e6c3b8d4',
    workflow: 'send-email',
    status: 'CANCELLED',
    duration: '4.0s',
  },
];

const RunsTable = SimpleTable<Run>;

const meta = {
  title: 'Molecules/SimpleTable',
  component: RunsTable,
  args: {
    data: runs,
    rowKey: (row) => row.id,
    columns: [
      {
        columnLabel: 'Run ID',
        cellRenderer: (row) => <span className="font-mono">{row.id}</span>,
      },
      { columnLabel: 'Workflow', cellRenderer: (row) => row.workflow },
      {
        columnLabel: 'Status',
        cellRenderer: (row) => (
          <Badge variant={statusVariant[row.status]}>{row.status}</Badge>
        ),
      },
      { columnLabel: 'Duration', cellRenderer: (row) => row.duration },
    ],
  },
  argTypes: {
    columns: { control: false },
    rowKey: { control: false },
  },
} satisfies Meta<typeof RunsTable>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Empty: Story = {
  args: {
    data: [],
  },
};
