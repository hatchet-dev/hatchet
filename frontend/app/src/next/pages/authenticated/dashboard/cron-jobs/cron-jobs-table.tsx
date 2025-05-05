import { Badge } from '@/next/components/ui/badge';
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
import { Button } from '@/next/components/ui/button';
import {
  MoreHorizontal,
  Trash2,
  CalendarDays,
  RefreshCw,
  Plus,
} from 'lucide-react';
import { CronsFilters, useCrons } from '@/next/hooks/use-crons';
import {
  PageSelector,
  PageSizeSelector,
  Pagination,
} from '@/next/components/ui/pagination';
import { Time } from '@/next/components/ui/time';
import { DestructiveDialog } from '@/next/components/ui/dialog/destructive-dialog';
import { useMemo, useState } from 'react';
import { CronWorkflows } from '@/lib/api';
import cronstrue from 'cronstrue';
import {
  FilterGroup,
  FilterKeyValue,
  FilterTaskSelect,
} from '@/next/components/ui/filters/filters';
import { AdditionalMetadata } from '@/next/components/ui/additional-meta';
import { DocsButton } from '@/next/components/ui/docs-button';
import docs from '@/next/lib/docs';

export default function CronJobsTable({
  onCreateClicked,
}: {
  onCreateClicked: () => void;
}) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedCron, setSelectedCron] = useState<CronWorkflows>();

  const {
    data: crons = [],
    isLoading,
    delete: deleteCron,
    filters: { filters, setFilter },
  } = useCrons();

  const handleDeleteCron = async (cronId: string) => {
    try {
      await deleteCron.mutateAsync(cronId);
      setDeleteDialogOpen(false);
    } catch (error) {
      console.error('Failed to delete cron job:', error);
    }
  };

  const additionalMetaOpts = useMemo(() => {
    if (!crons || crons.length === 0) {
      return [];
    }

    const allKeys = new Set<string>();
    crons.forEach((cron) => {
      if (cron.additionalMetadata) {
        Object.keys(cron.additionalMetadata).forEach((key) => allKeys.add(key));
      }
    });

    return Array.from(allKeys).map((key) => ({
      label: key,
      value: key,
    }));
  }, [crons]);

  if (isLoading) {
    return (
      <div className="flex justify-center items-center h-64">
        <RefreshCw className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <>
      <FilterGroup>
        <FilterTaskSelect<CronsFilters>
          name="workflowId"
          placeholder="Task Name"
        />
        <FilterKeyValue<CronsFilters>
          name="additionalMetadata"
          placeholder="Additional Metadata"
          options={additionalMetaOpts}
        />
      </FilterGroup>
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Workflow</TableHead>
              <TableHead>Expression</TableHead>
              <TableHead>Parsed</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Created</TableHead>
              <TableHead>Additional Metadata</TableHead>
              <TableHead className="text-right"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {crons.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} className="h-24">
                  <div className="flex flex-col items-center justify-center gap-4 py-8">
                    <p className="text-md">No cron jobs found.</p>
                    <p className="text-sm text-muted-foreground">
                      Create a new cron job to get started.
                    </p>
                    <Button onClick={onCreateClicked}>
                      <Plus className="h-4 w-4 mr-2" />
                      Create Cron Job
                    </Button>
                    <DocsButton doc={docs.home.cron_runs} />
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              crons.map((cron) => (
                <TableRow key={cron.metadata.id}>
                  <TableCell className="font-medium">
                    <div className="flex items-center">
                      <CalendarDays className="h-4 w-4 mr-2 text-muted-foreground" />
                      {cron.name || `Cron-${cron.metadata.id.substring(0, 8)}`}
                    </div>
                  </TableCell>
                  <TableCell>{cron.workflowName}</TableCell>
                  <TableCell>
                    <div className="flex flex-col">
                      <span className="text-xs text-muted-foreground">
                        {cron.cron}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col">
                      <span className="text-sm">
                        {cron.cron
                          ? cronstrue.toString(cron.cron)
                          : 'No schedule'}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    {cron.enabled ? (
                      <Badge
                        variant="outline"
                        className="bg-green-50 text-green-700 border-green-200"
                      >
                        Active
                      </Badge>
                    ) : (
                      <Badge
                        variant="outline"
                        className="bg-red-50 text-red-700 border-red-200"
                      >
                        Paused
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    <Time date={cron.metadata.createdAt} variant="timestamp" />
                  </TableCell>
                  <TableCell>
                    <AdditionalMetadata
                      metadata={cron.additionalMetadata}
                      onClick={(click) => {
                        setFilter('additionalMetadata', [
                          ...(filters.additionalMetadata || []),
                          `${click.key}:${click.value}`,
                        ]);
                      }}
                    />
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon">
                            <MoreHorizontal className="h-4 w-4" />
                            <span className="sr-only">Open menu</span>
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuLabel>Actions</DropdownMenuLabel>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            className="text-red-600"
                            onClick={(e) => {
                              e.stopPropagation();
                              setSelectedCron(cron);
                              setDeleteDialogOpen(true);
                            }}
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

        <DestructiveDialog
          open={deleteDialogOpen}
          onOpenChange={setDeleteDialogOpen}
          title="Delete Cron Job"
          description={`Are you sure you want to delete this cron job? This action will stop all future runs.`}
          confirmationText={selectedCron?.name || 'confirm'}
          confirmButtonText="Delete"
          onConfirm={() =>
            selectedCron && handleDeleteCron(selectedCron.metadata.id)
          }
        />
      </div>
      <Pagination className="py-4 justify-between flex flex-row">
        <PageSizeSelector />
        <PageSelector variant="dropdown" />
      </Pagination>
    </>
  );
}
