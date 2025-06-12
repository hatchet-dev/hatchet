import { ColumnDef } from '@tanstack/react-table';
import { Worker } from '@/lib/api';
import { WorkerStatusBadge } from './worker-status-badge';
import { SlotsBadge } from './worker-slots-badge';
import { WorkerId } from './worker-id';
import { Time } from '@/next/components/ui/time';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { Button } from '@/next/components/ui/button';
import { Play, Pause } from 'lucide-react';
import { Checkbox } from '@/next/components/ui/checkbox';

interface WorkerWithPoolInfo extends Worker {
  poolName: string;
}

export const createWorkerColumns = (
  handleWorkerClick: (workerId: string) => void,
  handleWorkerAction: (workerId: string, action: 'pause' | 'resume') => void,
): ColumnDef<WorkerWithPoolInfo>[] => [
  {
    id: 'select',
    header: ({ table }) => (
      <Checkbox
        checked={
          table.getIsAllPageRowsSelected() ||
          (table.getIsSomePageRowsSelected() && 'indeterminate')
        }
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all"
        className="translate-y-[2px]"
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label="Select row"
        className="translate-y-[2px]"
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'metadata.id',
    header: 'ID',
    cell: ({ row }) => (
      <button
        onClick={() => handleWorkerClick(row.original.metadata.id)}
        className="hover:underline text-left font-medium"
      >
        <WorkerId worker={row.original} poolName={row.original.poolName} />
      </button>
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => (
      <WorkerStatusBadge status={row.original.status} variant="outline" />
    ),
    filterFn: (row, id, value) => {
      // Handle multiple values as OR logic
      const statusValues = Array.isArray(value) ? value : [value];
      return statusValues.includes(row.getValue(id));
    },
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'slots',
    header: 'Slots',
    cell: ({ row }) => (
      <SlotsBadge
        available={
          row.original.status === 'ACTIVE' ? row.original.availableRuns || 0 : 0
        }
        max={row.original.maxRuns || 0}
      />
    ),
    enableSorting: false,
    enableHiding: true,
  },
  {
    accessorKey: 'lastHeartbeatAt',
    header: 'Last Heartbeat',
    cell: ({ row }) => (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <span>
              <Time date={row.original.lastHeartbeatAt} variant="timeSince" />
            </span>
          </TooltipTrigger>
          <TooltipContent>
            <Time date={row.original.lastHeartbeatAt} variant="timestamp" />
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    ),
    enableSorting: false,
    enableHiding: true,
  },
  {
    id: 'actions',
    header: 'Actions',
    cell: ({ row }) => (
      <div className="flex justify-end gap-2">
        {row.original.status !== 'ACTIVE' &&
          row.original.status !== 'INACTIVE' && (
            <Button
              variant="ghost"
              size="icon"
              onClick={() =>
                handleWorkerAction(row.original.metadata.id, 'resume')
              }
              title="Resume Worker"
            >
              <Play className="h-4 w-4" />
            </Button>
          )}
        {row.original.status !== 'PAUSED' &&
          row.original.status !== 'INACTIVE' && (
            <Button
              variant="ghost"
              size="icon"
              onClick={() =>
                handleWorkerAction(row.original.metadata.id, 'pause')
              }
              title="Pause Worker"
            >
              <Pause className="h-4 w-4" />
            </Button>
          )}
      </div>
    ),
    enableSorting: false,
    enableHiding: false,
  },
];

export type { WorkerWithPoolInfo };
