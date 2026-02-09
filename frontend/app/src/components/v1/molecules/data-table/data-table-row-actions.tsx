import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import {
  Tooltip,
  TooltipProvider,
  TooltipTrigger,
  TooltipContent,
} from '@/components/v1/ui/tooltip';
import { DotsVerticalIcon } from '@radix-ui/react-icons';

interface TableRowActionsProps<T> {
  row: T;
  actions?: {
    label: string;
    onClick: (data: T) => void;
    disabled?: boolean | string;
  }[];
}

export function TableRowActions<T>({ row, actions }: TableRowActionsProps<T>) {
  if (!actions?.length) {
    return null;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="sm"
          className="flex data-[state=open]:bg-muted"
        >
          <DotsVerticalIcon className="size-4" />
          <span className="sr-only">Open menu</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-[160px]">
        {actions?.map((action) => (
          <TooltipProvider key={action.label}>
            <Tooltip>
              <TooltipTrigger className="w-full">
                <DropdownMenuItem
                  onClick={() => action.onClick(row)}
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
