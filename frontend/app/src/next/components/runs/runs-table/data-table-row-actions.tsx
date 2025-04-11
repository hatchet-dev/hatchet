'use client';

import { Row } from '@tanstack/react-table';
import { MoreHorizontal, FileText, PlayCircle, StopCircle } from 'lucide-react';
import { Button } from '@/next/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/next/components/ui/dropdown-menu';
import { V1TaskSummary, V1TaskStatus } from '@/next/lib/api';
import { Link } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';

interface DataTableRowActionsProps<TData> {
  row: Row<TData>;
}

export function DataTableRowActions<TData>({
  row,
}: DataTableRowActionsProps<TData>) {
  const run = row.original as V1TaskSummary;
  const isRunning = run.status === ('RUNNING' as V1TaskStatus);

  return (
    <span className="group">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" className="h-8 w-8 p-0">
            <span className="sr-only">Open menu</span>
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem asChild>
            <Link to={ROUTES.runs.detail(run.metadata.id)}>
              <FileText className="mr-2 h-4 w-4" />
              View Details
            </Link>
          </DropdownMenuItem>
          {isRunning ? (
            <DropdownMenuItem>
              <StopCircle className="mr-2 h-4 w-4" />
              Cancel Run
            </DropdownMenuItem>
          ) : run.status !== ('CANCELLED' as V1TaskStatus) &&
            run.status !== ('FAILED' as V1TaskStatus) ? (
            <DropdownMenuItem>
              <PlayCircle className="mr-2 h-4 w-4" />
              Re-run
            </DropdownMenuItem>
          ) : null}
          <DropdownMenuSeparator />
          <DropdownMenuItem>
            <FileText className="mr-2 h-4 w-4" />
            View Logs
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </span>
  );
}
