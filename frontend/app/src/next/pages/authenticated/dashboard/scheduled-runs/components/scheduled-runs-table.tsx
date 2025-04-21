import { Button } from '@/next/components/ui/button';
import { Time } from '@/next/components/ui/time';
import { DocsButton } from '@/next/components/ui/docs-button';
import docs from '@/next/docs-meta-data';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/next/components/ui/table';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/next/components/ui/dropdown-menu';
import { MoreHorizontal, Trash2, Clock, Plus } from 'lucide-react';
import { ScheduledWorkflows } from '@/lib/api';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { useMemo, useState } from 'react';
import { DestructiveDialog } from '@/next/components/ui/dialog/destructive-dialog';
import {
  PageSelector,
  PageSizeSelector,
  Pagination,
  usePagination,
} from '@/next/components/ui/pagination';
import { useFilters } from '@/next/hooks/use-filters';
import useSchedules, { SchedulesFilters } from '@/next/hooks/use-schedules';
import {
  FilterGroup,
  FilterKeyValue,
  FilterTaskSelect,
} from '@/next/components/ui/filters/filters';
import { RunsBadge } from '@/next/components/runs/runs-badge';
import { AdditionalMetadata } from '@/next/components/ui/additional-meta';

interface ScheduledRunsTableProps {
  onCreateClicked: () => void;
}

export function ScheduledRunsTable({
  onCreateClicked,
}: ScheduledRunsTableProps) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedRun, setSelectedRun] = useState<ScheduledWorkflows | null>(
    null,
  );

  const paginationManager = usePagination();
  const { filters, setFilter } = useFilters<SchedulesFilters>();

  const { data: scheduledRunsData = [], delete: deleteSchedule } = useSchedules(
    {
      refetchInterval: 5000,
      paginationManager,
      filters,
    },
  );

  const handleDeleteClick = (run: ScheduledWorkflows) => {
    setSelectedRun(run);
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (selectedRun) {
      await deleteSchedule.mutateAsync(selectedRun.metadata.id);
      setDeleteDialogOpen(false);
      setSelectedRun(null);
    }
  };

  const additionalMetaOpts = useMemo(() => {
    if (!scheduledRunsData || scheduledRunsData.length === 0) {
      return [];
    }

    const allKeys = new Set<string>();
    scheduledRunsData.forEach((run) => {
      if (run.additionalMetadata) {
        Object.keys(run.additionalMetadata).forEach((key) => allKeys.add(key));
      }
    });

    return Array.from(allKeys).map((key) => ({
      label: key,
      value: key,
    }));
  }, [scheduledRunsData]);

  return (
    <>
      <FilterGroup>
        <FilterTaskSelect<SchedulesFilters>
          name="workflowId"
          placeholder="Task Name"
          multi
        />
        <FilterKeyValue<SchedulesFilters>
          name="additionalMetadata"
          placeholder="Additional Metadata"
          options={additionalMetaOpts}
        />
      </FilterGroup>
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Task Name</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Trigger At</TableHead>
              <TableHead>Created At</TableHead>
              <TableHead className="text-right"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {scheduledRunsData.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="h-24">
                  <div className="flex flex-col items-center justify-center gap-4 py-8">
                    <p className="text-md">No scheduled runs found.</p>
                    <p className="text-sm text-muted-foreground">
                      Create a new scheduled run to get started.
                    </p>
                    {
                      <Button onClick={onCreateClicked}>
                        <Plus className="h-4 w-4 mr-2" />
                        Create Scheduled Run
                      </Button>
                    }
                    <DocsButton doc={docs.home['scheduled-runs']} />
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              scheduledRunsData.map((run) => (
                <TableRow key={run.metadata.id} className="cursor-pointer">
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{run.workflowName}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <RunsBadge status={run.workflowRunStatus} />
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Clock className="h-4 w-4 text-muted-foreground" />
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger>
                            <Time date={run.triggerAt} variant="timeSince" />
                          </TooltipTrigger>
                          <TooltipContent>
                            <Time date={run.triggerAt} variant="timestamp" />
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Time date={run.metadata.createdAt} variant="timestamp" />
                    </div>
                  </TableCell>
                  <TableCell>
                    <AdditionalMetadata
                      metadata={run.additionalMetadata}
                      onClick={(click) => {
                        setFilter('additionalMetadata', [
                          ...(filters.additionalMetadata || []),
                          `${click.key}:${click.value}`,
                        ]);
                      }}
                    />
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-2">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={(e) => e.stopPropagation()}
                          >
                            <MoreHorizontal className="h-4 w-4" />
                            <span className="sr-only">Open menu</span>
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuLabel>Actions</DropdownMenuLabel>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDeleteClick(run);
                            }}
                            className="text-red-600"
                          >
                            <Trash2 className="h-4 w-4 mr-2" />
                            Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
      <Pagination className="py-2 justify-between flex flex-row">
        <PageSizeSelector />
        <PageSelector variant="dropdown" />
      </Pagination>
      <DestructiveDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        title="Delete Scheduled Run"
        description={`Are you sure you want to delete the scheduled run "${selectedRun?.workflowName}"? This will stop all future runs.`}
        confirmationText="confirm"
        confirmButtonText="Delete"
        onConfirm={handleDeleteConfirm}
      />
    </>
  );
}
