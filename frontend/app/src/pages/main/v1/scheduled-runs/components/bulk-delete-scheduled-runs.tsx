import { useToast } from '@/components/v1/hooks/use-toast';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { Input } from '@/components/v1/ui/input';
import { Spinner } from '@/components/v1/ui/loading';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api from '@/lib/api';
import { ScheduledWorkflowsBulkDeleteFilter } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { useEffect, useRef, useState } from 'react';

export function BulkDeleteScheduledRuns({
  open,
  scheduledRunIds,
  filter,
  onOpenChange,
  onSuccess,
}: {
  open: boolean;
  scheduledRunIds: string[];
  filter?: ScheduledWorkflowsBulkDeleteFilter;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const { toast } = useToast();
  const { handleApiError } = useApiError({});
  const formatCount = (n: number) => new Intl.NumberFormat().format(n);

  const isFilterMode = scheduledRunIds.length === 0 && !!filter;
  const [filterCount, setFilterCount] = useState<number | undefined>(undefined);
  const [isFilterCountLoading, setIsFilterCountLoading] = useState(false);
  const [filterCountError, setFilterCountError] = useState<string | undefined>(
    undefined,
  );
  const [lastResult, setLastResult] = useState<
    | {
        deletedIds: string[];
        errors: { id?: string; error: string }[];
      }
    | undefined
  >(undefined);
  const [confirmationText, setConfirmationText] = useState('');
  const expectedConfirmation = 'DELETE';
  const isConfirmed =
    confirmationText.trim().toLowerCase() ===
    expectedConfirmation.toLowerCase();
  const [progress, setProgress] = useState<{
    phase: 'idle' | 'collecting' | 'deleting' | 'done';
    processed: number;
    total?: number;
  }>({ phase: 'idle', processed: 0 });

  useEffect(() => {
    if (!open) {
      setConfirmationText('');
      setProgress({ phase: 'idle', processed: 0 });
    }

    if (!open || !isFilterMode || !filter) {
      setFilterCount(undefined);
      setIsFilterCountLoading(false);
      setFilterCountError(undefined);
      setLastResult(undefined);
      return;
    }

    let cancelled = false;
    setIsFilterCountLoading(true);
    setFilterCountError(undefined);
    setFilterCount(undefined);
    setLastResult(undefined);

    const limit = 200;

    const load = async () => {
      const first = await api.workflowScheduledList(tenantId, {
        ...filter,
        limit,
        offset: 0,
      });

      const numPages = first.data?.pagination?.num_pages ?? 0;
      const firstLen = first.data?.rows?.length ?? 0;

      if (numPages <= 1) {
        if (!cancelled) {
          setFilterCount(firstLen);
        }
        return;
      }

      const lastOffset = (numPages - 1) * limit;
      const last = await api.workflowScheduledList(tenantId, {
        ...filter,
        limit,
        offset: lastOffset,
      });
      const lastLen = last.data?.rows?.length ?? 0;

      if (!cancelled) {
        setFilterCount((numPages - 1) * limit + lastLen);
      }
    };

    load()
      .catch((e) => {
        if (!cancelled) {
          setFilterCountError((e as Error).message);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setIsFilterCountLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [open, isFilterMode, filter, tenantId]);

  const deleteMutation = useMutation({
    mutationKey: [
      'scheduled-run:bulk-delete',
      tenantId,
      scheduledRunIds,
      filter,
    ],
    mutationFn: async () => {
      const allDeletedIds: string[] = [];
      const allErrors: { id?: string; error: string }[] = [];

      const deleteChunk = async (ids: string[]) => {
        if (ids.length === 0) {
          return;
        }
        const res = await api.workflowScheduledBulkDelete(tenantId, {
          scheduledWorkflowRunIds: ids,
        });
        allDeletedIds.push(...(res.data?.deletedIds ?? []));
        allErrors.push(...(res.data?.errors ?? []));
      };

      // Important: in filter/all mode we must NOT delete while paginating with `offset`,
      // otherwise the dataset shifts and we can skip rows. Instead, collect ids first.
      let idsToDelete: string[] = [];

      if (scheduledRunIds.length > 0) {
        idsToDelete = scheduledRunIds;
      } else {
        const limit = 1000;
        let offset = 0;
        const effectiveFilter = filter ?? {};

        setProgress({
          phase: 'collecting',
          processed: 0,
          total: filterCount,
        });

        let hasMore = true;
        while (hasMore) {
          const listRes = await api.workflowScheduledList(tenantId, {
            ...effectiveFilter,
            limit,
            offset,
          });

          const rows = listRes.data?.rows ?? [];
          const ids = rows
            .map((r) => r?.metadata?.id)
            .filter((id): id is string => !!id);

          if (ids.length === 0) {
            hasMore = false;
            continue;
          }

          idsToDelete.push(...ids);
          setProgress((p) => ({
            ...p,
            processed: p.processed + ids.length,
          }));

          const numPages = listRes.data?.pagination?.num_pages;
          const pageIndex = Math.floor(offset / limit) + 1;
          hasMore = false;
          if (numPages != null) {
            if (pageIndex < numPages) {
              offset += rows.length;
              hasMore = true;
            }
          } else if (rows.length === limit) {
            offset += rows.length;
            hasMore = true;
          }
        }
      }

      setProgress({
        phase: 'deleting',
        processed: 0,
        total: idsToDelete.length,
      });

      const batchSize = 500;
      for (let i = 0; i < idsToDelete.length; i += batchSize) {
        const chunk = idsToDelete.slice(i, i + batchSize);
        await deleteChunk(chunk);
        setProgress((p) => ({
          ...p,
          processed: p.processed + chunk.length,
        }));
      }

      setProgress((p) => ({ ...p, phase: 'done' }));
      return { deletedIds: allDeletedIds, errors: allErrors };
    },
    onSuccess: (data) => {
      const deleted = data?.deletedIds?.length || 0;
      const errors = data?.errors?.length || 0;
      const description = `Deleted ${formatCount(deleted)}. ${
        errors ? `${formatCount(errors)} failed.` : ''
      }`.trim();

      toast({
        title: `Delete scheduled runs`,
        description,
      });

      setLastResult({
        deletedIds: data?.deletedIds ?? [],
        errors: (data?.errors ?? []) as { id?: string; error: string }[],
      });

      onSuccess();
      if (!errors) {
        onOpenChange(false);
      }
    },
    onError: handleApiError,
  });

  const submitLockRef = useRef(false);
  const handleDelete = () => {
    if (submitLockRef.current || deleteMutation.isPending) {
      return;
    }

    submitLockRef.current = true;
    deleteMutation.mutate(undefined, {
      onSettled: () => {
        submitLockRef.current = false;
      },
    });
  };

  return (
    <ConfirmDialog
      isOpen={open}
      title="Delete scheduled runs"
      submitLabel="Delete"
      submitVariant="destructive"
      submitDisabled={(scheduledRunIds.length === 0 && !filter) || !isConfirmed}
      isLoading={deleteMutation.isPending}
      onCancel={() => onOpenChange(false)}
      onSubmit={handleDelete}
      description={
        <div className="space-y-4">
          {deleteMutation.isError && (
            <div className="rounded-md border border-destructive/50 bg-destructive/5 p-3 text-sm text-destructive">
              {(deleteMutation.error as Error)?.message || 'Failed to delete.'}
            </div>
          )}

          {lastResult?.errors?.length ? (
            <div className="rounded-md border border-destructive/50 bg-destructive/5 p-3 text-sm text-destructive">
              <div className="font-medium">
                {lastResult.errors.length} failed to delete.
              </div>
              <div className="mt-2 space-y-1 text-xs">
                {lastResult.errors.slice(0, 10).map((e, idx) => (
                  <div
                    key={`${e.id ?? 'unknown'}-${idx}`}
                    className="break-all"
                  >
                    {e.id ? `${e.id}: ` : ''}
                    {e.error}
                  </div>
                ))}
                {lastResult.errors.length > 10 && (
                  <div>…and {lastResult.errors.length - 10} more.</div>
                )}
              </div>
            </div>
          ) : null}

          {scheduledRunIds.length > 0 ? (
            <p>
              You are about to delete <b>{scheduledRunIds.length}</b> scheduled
              run{scheduledRunIds.length === 1 ? '' : 's'}. This cannot be
              undone.
            </p>
          ) : (
            <div className="space-y-2">
              <p>
                You are about to delete all scheduled runs matching the current
                filters. This cannot be undone.
              </p>
              <div className="text-xs text-muted-foreground">
                {isFilterCountLoading && (
                  <span className="inline-flex items-center gap-2">
                    <Spinner /> Counting matching runs…
                  </span>
                )}
                {!isFilterCountLoading && filterCountError && (
                  <span className="text-destructive">{filterCountError}</span>
                )}
                {!isFilterCountLoading &&
                  !filterCountError &&
                  filterCount != null && (
                    <span>
                      Affected scheduled runs: <b>{formatCount(filterCount)}</b>
                    </span>
                  )}
              </div>
            </div>
          )}

          {progress.phase === 'collecting' && (
            <div className="text-xs text-muted-foreground">
              Collecting {formatCount(progress.processed)}
              {progress.total != null
                ? ` / ${formatCount(progress.total)}`
                : ''}{' '}
              ids…
            </div>
          )}

          {progress.phase === 'deleting' && (
            <div className="text-xs text-muted-foreground">
              Deleting {formatCount(progress.processed)}
              {progress.total != null
                ? ` / ${formatCount(progress.total)}`
                : ''}
              …
            </div>
          )}

          {!deleteMutation.isPending &&
            progress.phase !== 'collecting' &&
            progress.phase !== 'deleting' && (
              <div className="space-y-2">
                <div className="text-xs text-muted-foreground">
                  Type <span className="font-mono">{expectedConfirmation}</span>{' '}
                  to confirm.
                </div>
                <Input
                  type="text"
                  value={confirmationText}
                  onChange={(e) => setConfirmationText(e.target.value)}
                  placeholder={`Type ${expectedConfirmation} to confirm`}
                />
              </div>
            )}
        </div>
      }
    />
  );
}
