import { strategyRef } from './components/concurrency-format';
import { ScopeFilter, StrategiesPanel } from './components/strategies-panel';
import { TotalsStrip } from './components/totals-strip';
import {
  STRATEGIES_PAGE_SIZE,
  useConcurrencyStrategies,
} from './hooks/use-concurrency';
import { ConcurrencySearch, normalizeConcurrencySearch } from './search';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { useLiveClock } from '@/hooks/use-live-clock';
import { useSidePanel } from '@/hooks/use-side-panel';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { V1ConcurrencyStrategy } from '@/lib/api';
import { docsPages } from '@/lib/generated/docs';
import { tenantConcurrencyRoute } from '@/router';
import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useMemo, useRef } from 'react';

export default function ConcurrencyPage() {
  const { tenantId } = useCurrentTenantId();

  // Keyed by tenant so a tenant switch resets the selection and filters.
  return <Concurrency key={tenantId} tenantId={tenantId} />;
}

type ConcurrencyProps = { tenantId: string };

function Concurrency({ tenantId }: ConcurrencyProps) {
  // ticks so elapsed-time cells stay live between refetches
  const now = useLiveClock(true, 1000);
  const queryClient = useQueryClient();
  const { data, isLoading, isRefetching, error, refetch } =
    useConcurrencyStrategies(tenantId);

  const refresh = () => {
    void refetch();
    void queryClient.invalidateQueries({
      queryKey: ['v1:concurrency:keys', tenantId],
    });
  };

  const view = tenantConcurrencyRoute.useSearch();
  const navigate = tenantConcurrencyRoute.useNavigate();

  const scope: ScopeFilter = view.scope ?? 'ALL';
  const selectedRef = view.strategy;

  const updateView = (patch: ConcurrencySearch) =>
    void navigate({
      search: normalizeConcurrencySearch({ ...view, ...patch }),
      replace: true,
    });

  const sidePanel = useSidePanel();

  const all = useMemo(() => data?.rows ?? [], [data]);

  const visible = useMemo(
    () => all.filter((s) => scope === 'ALL' || s.scope === scope),
    [all, scope],
  );

  const numPages = Math.ceil(visible.length / STRATEGIES_PAGE_SIZE);
  // clamped so a page emptied by strategies going away shows the last page instead
  const page = Math.min((view.page ?? 1) - 1, Math.max(numPages - 1, 0));
  const pageRows = visible.slice(
    page * STRATEGIES_PAGE_SIZE,
    (page + 1) * STRATEGIES_PAGE_SIZE,
  );

  const openKeys = (ref: string, initialSearch?: string) =>
    sidePanel.open({
      type: 'concurrency-keys',
      content: { tenantId, strategy: ref, initialSearch },
    });

  // A filter that hides the selected strategy also clears it and closes its keys, so the
  // URL never names a strategy other than the one shown.
  const setScope = (s: ScopeFilter) => {
    const selectedStrategy = all.find((x) => strategyRef(x) === selectedRef);
    const keep = s === 'ALL' || selectedStrategy?.scope === s;

    updateView({
      scope: s === 'ALL' ? undefined : s,
      strategy: keep ? selectedRef : undefined,
      page: undefined,
    });

    if (!keep && sidePanel.isOpen) {
      sidePanel.close();
    }
  };

  const selectStrategy = (s: V1ConcurrencyStrategy) => {
    updateView({ strategy: strategyRef(s) });
    openKeys(strategyRef(s));
  };

  // clears the filters and turns to the strategy's page so it is always visible
  const openKey = (s: V1ConcurrencyStrategy, key: string) => {
    const index = all.findIndex((x) => strategyRef(x) === strategyRef(s));

    updateView({
      scope: undefined,
      strategy: strategyRef(s),
      page: Math.floor(Math.max(index, 0) / STRATEGIES_PAGE_SIZE) + 1,
    });
    openKeys(strategyRef(s), key);
  };

  // a link carrying ?strategy= opens that strategy's keys once the strategies load
  const restored = useRef(false);

  useEffect(() => {
    if (restored.current || isLoading) {
      return;
    }

    restored.current = true;

    if (selectedRef && all.some((s) => strategyRef(s) === selectedRef)) {
      sidePanel.open({
        type: 'concurrency-keys',
        content: { tenantId, strategy: selectedRef },
      });
    }
  }, [isLoading, all, selectedRef, sidePanel, tenantId]);

  // closing the side panel clears the selection
  const wasOpen = useRef(sidePanel.isOpen);

  useEffect(() => {
    if (wasOpen.current && !sidePanel.isOpen) {
      void navigate({
        search: (prev) =>
          normalizeConcurrencySearch({ ...prev, strategy: undefined }),
        replace: true,
      });
    }

    wasOpen.current = sidePanel.isOpen;
  }, [sidePanel.isOpen, navigate]);

  return (
    <div className="flex min-w-0 flex-col gap-3">
      <div>
        <h1 className="text-lg font-semibold">Concurrency</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Every concurrency strategy in this tenant, the keys it is tracking
          right now, and how many runs are queued on them.
        </p>
      </div>

      {error ? (
        <div className="rounded-lg px-4 py-8 text-center text-sm text-destructive ring-1 ring-border/50">
          Could not load concurrency strategies.
        </div>
      ) : !isLoading && all.length === 0 ? (
        <div className="rounded-lg px-4 py-16 ring-1 ring-border/50">
          <EmptyState
            title="No concurrency strategies"
            description="Concurrency strategies limit how many runs can execute at once."
            docPage={docsPages.v1.concurrency}
            docLabel="Learn about concurrency"
          />
        </div>
      ) : (
        <>
          <TotalsStrip
            strategies={all}
            loading={isLoading}
            now={now}
            onOpenKey={openKey}
          />

          <StrategiesPanel
            strategies={pageRows}
            loading={isLoading}
            selectedRef={selectedRef}
            onSelect={(s) => selectStrategy(s)}
            scope={scope}
            onScopeChange={setScope}
            page={page}
            numPages={numPages}
            total={visible.length}
            onPageChange={(p) => updateView({ page: p + 1 })}
            isRefetching={isRefetching}
            onRefetch={refresh}
            now={now}
          />
        </>
      )}
    </div>
  );
}
