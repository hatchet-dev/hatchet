import { useConcurrencyStrategies } from '../hooks/use-concurrency';
import { strategyRef } from './concurrency-format';
import { KeysPanel } from './keys-panel';
import { useLiveClock } from '@/hooks/use-live-clock';
import { useState } from 'react';

export type KeysSidePanelProps = {
  tenantId: string;
  // strategyRef of the strategy to show
  strategy: string;
  initialSearch?: string;
};

// Rendered by the app's side panel, outside the page, so it keeps its own filter state
// and reads the strategy's live counts from the cached strategies query.
export function KeysSidePanel({
  tenantId,
  strategy: ref,
  initialSearch = '',
}: KeysSidePanelProps) {
  const now = useLiveClock(true, 1000);
  const { data, isLoading } = useConcurrencyStrategies(tenantId);

  const [search, setSearch] = useState(initialSearch);
  const [onlyQueued, setOnlyQueued] = useState(false);
  const [page, setPage] = useState(0);

  const strategy = data?.rows.find((s) => strategyRef(s) === ref);

  if (!strategy) {
    return (
      <div className="py-8 text-center text-sm text-muted-foreground">
        {isLoading ? 'Loading…' : 'This strategy no longer exists.'}
      </div>
    );
  }

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <KeysPanel
        tenantId={tenantId}
        strategy={strategy}
        search={search}
        onSearchChange={(q) => {
          setSearch(q);
          setPage(0);
        }}
        onlyQueued={onlyQueued}
        onOnlyQueuedChange={(v) => {
          setOnlyQueued(v);
          setPage(0);
        }}
        page={page}
        onPageChange={setPage}
        now={now}
      />
    </div>
  );
}
