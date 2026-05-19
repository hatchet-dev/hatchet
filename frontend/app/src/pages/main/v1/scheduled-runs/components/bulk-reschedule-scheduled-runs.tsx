import { useToast } from '@/components/v1/hooks/use-toast';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { Input } from '@/components/v1/ui/input';
import { Spinner } from '@/components/v1/ui/loading';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, { ScheduledWorkflows } from '@/lib/api';
import { ScheduledWorkflowsBulkDeleteFilter } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { useEffect, useMemo, useRef, useState } from 'react';

export function BulkRescheduleScheduledRuns({
  open,
  scheduledRunIds,
  filter,
  initialRuns,
  onOpenChange,
  onSuccess,
}: {
  open: boolean;
  scheduledRunIds: string[];
  filter?: ScheduledWorkflowsBulkDeleteFilter;
  initialRuns: ScheduledWorkflows[];
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const { toast } = useToast();
  const { handleApiError } = useApiError({});
  const formatCount = (n: number) => new Intl.NumberFormat().format(n);

  const initialById = useMemo(() => {
    const m = new Map<string, ScheduledWorkflows>();
    for (const r of initialRuns) {
      m.set(r.metadata.id, r);
    }
    return m;
  }, [initialRuns]);

  const [targetDate, setTargetDate] = useState<Date | undefined>(undefined);
  const [filterCount, setFilterCount] = useState<number | undefined>(undefined);
  const [isFilterCountLoading, setIsFilterCountLoading] = useState(false);
  const [filterCountError, setFilterCountError] = useState<string | undefined>(
    undefined,
  );
  const [lastResult, setLastResult] = useState<
    | {
        updatedIds: string[];
        errors: { id?: string; error: string }[];
      }
    | undefined
  >(undefined);
  const [confirmationText, setConfirmationText] = useState('');
  const expectedConfirmation = 'RESCHEDULE';
  const isConfirmed =
    confirmationText.trim().toLowerCase() ===
    expectedConfirmation.toLowerCase();
  const [progress, setProgress] = useState<{
    phase: 'idle' | 'running' | 'done';
    processed: number;
    total?: number;
  }>({ phase: 'idle', processed: 0 });

  const isFilterMode = scheduledRunIds.length === 0 && !!filter;

  useEffect(() => {
    if (!open) {
      setConfirmationText('');
      setProgress({ phase: 'idle', processed: 0 });
    }
  }, [open]);

  // In filter mode, fetch an exact count quickly using 1-2 list queries (first page -> num_pages, then last page).
  useEffect(() => {
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

  // When opening, default the target date to the first available trigger time (if present).
  useEffect(() => {
    if (!open) {
      return;
    }

    if (targetDate) {
      return;
    }

    // Try to find a trigger time in already-loaded page data first.
    for (const id of scheduledRunIds) {
      const triggerAt = initialById.get(id)?.triggerAt;
      if (triggerAt) {
        setTargetDate(new Date(triggerAt));
        return;
      }
    }

    let cancelled = false;

    const loadDefault = async () => {
      try {
        if (isFilterMode && filter) {
          const res = await api.workflowScheduledList(tenantId, {
            ...filter,
            limit: 1,
            offset: 0,
          });
          const first = res.data?.rows?.[0];
          if (!cancelled && first?.triggerAt) {
            setTargetDate(new Date(first.triggerAt));
          }
          return;
        }

        // Selected mode: fall back to fetching the first id if it's not in initial data.
        const firstId = scheduledRunIds[0];
        if (!firstId) {
          return;
        }
        const res = await api.workflowScheduledGet(tenantId, firstId);
        if (!cancelled && res.data?.triggerAt) {
          setTargetDate(new Date(res.data.triggerAt));
        }
      } catch {
        // Ignore default load errors; the user can still pick a time manually.
      }
    };

    loadDefault();

    return () => {
      cancelled = true;
    };
  }, [
    open,
    targetDate,
    scheduledRunIds,
    initialById,
    isFilterMode,
    filter,
    tenantId,
  ]);

  const selectedUpdates = useMemo(() => {
    if (!targetDate) {
      return [];
    }

    const triggerAt = targetDate.toISOString();
    return scheduledRunIds.map((id) => ({ id, triggerAt }));
  }, [scheduledRunIds, targetDate]);

  const updateMutation = useMutation({
    mutationKey: [
      'scheduled-run:bulk-update',
      tenantId,
      scheduledRunIds,
      filter,
    ],
    mutationFn: async () => {
      if (!targetDate) {
        return {
          updatedIds: [],
          errors: [] as { id?: string; error: string }[],
        };
      }

      setProgress({
        phase: 'running',
        processed: 0,
        total:
          scheduledRunIds.length > 0 ? scheduledRunIds.length : filterCount,
      });

      const allUpdatedIds: string[] = [];
      const allErrors: { id?: string; error: string }[] = [];

      if (isFilterMode && filter) {
        const limit = 1000;
        let offset = 0;
        const triggerAt = targetDate.toISOString();

        let hasMore = true;
        while (hasMore) {
          const listRes = await api.workflowScheduledList(tenantId, {
            ...filter,
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

          const batchSize = 500;
          for (let i = 0; i < ids.length; i += batchSize) {
            const chunkIds = ids.slice(i, i + batchSize);
            const chunkUpdates = chunkIds.map((id) => ({ id, triggerAt }));
            const res = await api.workflowScheduledBulkUpdate(tenantId, {
              updates: chunkUpdates,
            });
            allUpdatedIds.push(...(res.data?.updatedIds ?? []));
            allErrors.push(...(res.data?.errors ?? []));
            setProgress((p) => ({
              ...p,
              processed: p.processed + chunkIds.length,
            }));
          }

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
      } else {
        const batchSize = 500;
        for (let i = 0; i < selectedUpdates.length; i += batchSize) {
          const chunk = selectedUpdates.slice(i, i + batchSize);
          const res = await api.workflowScheduledBulkUpdate(tenantId, {
            updates: chunk,
          });
          allUpdatedIds.push(...(res.data?.updatedIds ?? []));
          allErrors.push(...(res.data?.errors ?? []));
          setProgress((p) => ({
            ...p,
            processed: p.processed + chunk.length,
          }));
        }
      }

      setProgress((p) => ({ ...p, phase: 'done' }));
      return { updatedIds: allUpdatedIds, errors: allErrors };
    },
    onSuccess: (data) => {
      const updated = data?.updatedIds?.length || 0;
      const errors = data?.errors?.length || 0;
      const description = `Updated ${formatCount(updated)}. ${
        errors ? `${formatCount(errors)} failed.` : ''
      }`.trim();

      toast({
        title: `Reschedule scheduled runs`,
        description,
      });

      setLastResult({
        updatedIds: data?.updatedIds ?? [],
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
  const handleReschedule = () => {
    if (submitLockRef.current || updateMutation.isPending) {
      return;
    }

    submitLockRef.current = true;
    updateMutation.mutate(undefined, {
      onSettled: () => {
        submitLockRef.current = false;
      },
    });
  };

  return (
    <ConfirmDialog
      isOpen={open}
      title="Reschedule scheduled runs"
      submitLabel="Reschedule"
      submitVariant="destructive"
      isLoading={updateMutation.isPending}
      submitDisabled={
        !targetDate ||
        !isConfirmed ||
        (isFilterMode &&
          (isFilterCountLoading ||
            !!filterCountError ||
            !filterCount ||
            filterCount <= 0)) ||
        (!isFilterMode && scheduledRunIds.length === 0) ||
        (!isFilterMode && selectedUpdates.length !== scheduledRunIds.length)
      }
      onCancel={() => onOpenChange(false)}
      onSubmit={handleReschedule}
      description={
        <div className="space-y-3">
          {updateMutation.isError && (
            <div className="rounded-md border border-destructive/50 bg-destructive/5 p-3 text-sm text-destructive">
              {(updateMutation.error as Error)?.message ||
                'Failed to reschedule.'}
            </div>
          )}

          {lastResult?.errors?.length ? (
            <div className="rounded-md border border-destructive/50 bg-destructive/5 p-3 text-sm text-destructive">
              <div className="font-medium">
                {lastResult.errors.length} failed to reschedule.
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

          <div className="rounded-md border p-3">
            <div className="flex flex-row items-center justify-between gap-4">
              <div className="min-w-0">
                <div className="text-sm font-medium">New trigger time</div>
                <div className="text-xs text-muted-foreground">
                  {isFilterMode
                    ? 'This will apply to all scheduled runs matching the current filters.'
                    : 'This will apply to all selected scheduled runs.'}
                </div>
              </div>

              <DateTimePicker
                date={targetDate}
                setDate={setTargetDate}
                label="Trigger at"
              />
            </div>
          </div>

          <div className="rounded-md border p-3">
            <div className="flex flex-row items-center justify-between gap-4">
              <div className="min-w-0">
                <div className="text-sm font-medium">
                  Affected scheduled runs:{' '}
                  {isFilterMode
                    ? filterCount != null
                      ? formatCount(filterCount)
                      : '—'
                    : formatCount(scheduledRunIds.length)}
                </div>
                <div className="text-xs text-muted-foreground">
                  {isFilterMode ? (
                    <>
                      {isFilterCountLoading && (
                        <span>Counting matching runs… </span>
                      )}
                      {!isFilterCountLoading && filterCountError && (
                        <span className="text-destructive">
                          {filterCountError}
                        </span>
                      )}
                      {!isFilterCountLoading &&
                        !filterCountError &&
                        filterCount != null && (
                          <span>Ready to reschedule.</span>
                        )}
                    </>
                  ) : (
                    <span>Ready to reschedule.</span>
                  )}
                </div>
              </div>

              {isFilterCountLoading && <Spinner />}
            </div>
          </div>

          {progress.phase === 'running' && (
            <div className="text-xs text-muted-foreground">
              Rescheduling {formatCount(progress.processed)}
              {progress.total != null
                ? ` / ${formatCount(progress.total)}`
                : ''}
              …
            </div>
          )}

          {!updateMutation.isPending && progress.phase !== 'running' && (
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
