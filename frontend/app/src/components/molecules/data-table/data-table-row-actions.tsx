import { DotsVerticalIcon } from '@radix-ui/react-icons';
import { Row } from '@tanstack/react-table';

import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

import { IDGetter } from './data-table';
import {
  Tooltip,
  TooltipProvider,
  TooltipTrigger,
  TooltipContent,
} from '@/components/ui/tooltip';

interface DataTableRowActionsProps<TData extends IDGetter<TData>> {
  row: Row<TData>;
  actions?: {
    label: string;
    onClick: (data: TData) => void;
    disabled?: boolean | string;
  }[];
}

export function DataTableRowActions<TData extends IDGetter<TData>>({
  row,
  actions,
}: DataTableRowActionsProps<TData>) {
  if (!actions?.length) {
    return null;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          className="flex h-8 w-8 p-0 data-[state=open]:bg-muted"
        >
          <DotsVerticalIcon className="h-4 w-4" />
          <span className="sr-only">Open menu</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-[160px]">
        {actions?.map((action) => (
          <TooltipProvider key={action.label}>
            <Tooltip>
              <TooltipTrigger className="w-full">
                <DropdownMenuItem
                  onClick={() => action.onClick(row.original)}
                  disabled={!!action.disabled}
                  className="w-full hover:cursor-pointer"
                >
                  {action.label}
                </DropdownMenuItem>
              </TooltipTrigger>
              {action.disabled && (
                <TooltipContent>{action.disabled}</TooltipContent>
              )}
            </Tooltip>
          </TooltipProvider>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
