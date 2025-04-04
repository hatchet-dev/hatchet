import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { cn } from '@/lib/utils';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';

interface WorkerLabel {
  key: string;
  intValue: number;
  strValue: string;
}

interface DesiredWorkerLabel extends WorkerLabel {
  is_true: boolean;
  weight: number;
  comparator: string;
  required: boolean;
}

export interface SemaphoreEventData {
  desired_worker_labels?: DesiredWorkerLabel[];
  actual_worker_labels?: WorkerLabel[];
}

export const mapSemaphoreExtra = (data: SemaphoreEventData) => {
  return (data.desired_worker_labels || []).map((label) => {
    const actual = data.actual_worker_labels?.find(
      (actual) => actual.key === label.key,
    );
    return {
      metadata: { id: 'key', createdAt: new Date(), updatedAt: new Date() }, // Hack to make the table work
      key: label.key,
      desired: label.intValue || label.strValue,
      actual: actual?.intValue || actual?.strValue || 'N/A',
      comparator: label.comparator,
      weight: label.weight,
      is_true: label.is_true,
      required: label.required,
    };
  });
};

const COMPARATOR_MAP: { [key: string]: string } = {
  EQUAL: '==',
  NOT_EQUAL: '!=',
  GREATER_THAN: '>',
  LESS_THAN: '<',
  GREATER_THAN_OR_EQUAL: '>=',
  LESS_THAN_OR_EQUAL: '<=',
};

export type SemaphoreExtra = ReturnType<typeof mapSemaphoreExtra>[0];

const LABEL_STATUS_VARIANTS: Record<string, string> = {
  true: 'border-transparent rounded-full bg-green-500',
  fail_required: 'border-transparent rounded-full bg-red-500',
  fail_not_required: 'border-transparent rounded-full bg-yellow-500',
};

function LabelStatus({
  is_true,
  required,
}: {
  is_true: boolean;
  required: boolean;
}) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>
          <div
            className={cn(
              LABEL_STATUS_VARIANTS[
                is_true
                  ? 'true'
                  : required
                    ? 'fail_required'
                    : 'fail_not_required'
              ],
              'rounded-full h-[6px] w-[6px]',
            )}
          />
        </TooltipTrigger>
        <TooltipContent>
          {is_true
            ? 'Actual value matches desired value'
            : required
              ? 'Actual value does not match desired value and is required'
              : 'Actual value does not match desired value but is not required'}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export const semaphoreExtraColumns: ColumnDef<SemaphoreExtra>[] = [
  {
    accessorKey: 'key',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Key" />
    ),
    cell: ({ row }) => {
      return (
        <div className="pl-1 min-w-fit whitespace-nowrap bold flex-row flex gap-2 items-center">
          <LabelStatus
            is_true={row.original.is_true}
            required={row.original.required}
          />
          {row.original.key}
        </div>
      );
    },
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'desired',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Desired" />
    ),
    cell: ({ row }) => {
      return (
        <div className="pl-0 min-w-fit whitespace-nowrap bold">
          {row.original.desired}
        </div>
      );
    },
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'comparator',
    header: () => <></>,
    cell: ({ row }) => {
      return (
        <div className="pl-0 min-w-fit whitespace-nowrap bold">
          {row.original.comparator in COMPARATOR_MAP
            ? COMPARATOR_MAP[row.original.comparator]
            : row.original.comparator}
        </div>
      );
    },
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'actual',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Actual" />
    ),
    cell: ({ row }) => {
      return (
        <div className="pl-0 min-w-fit whitespace-nowrap bold">
          {row.original.actual}
        </div>
      );
    },
    enableSorting: false,
    enableHiding: false,
  },
];
