import { ActiveRunsBeforeWindow } from '../hooks/use-active-runs-before-window';
import { Button } from '@/components/v1/ui/button';
import { Info } from 'lucide-react';

export function ActiveRunsBeforeWindowNote({
  activeRuns,
}: {
  activeRuns: ActiveRunsBeforeWindow;
}) {
  const { count, capped, showActiveRuns } = activeRuns;

  if (count === 0) {
    return null;
  }

  const countLabel = `${count.toLocaleString()}${capped ? '+' : ''}`;
  const isSingular = count === 1 && !capped;

  return (
    <div className="flex flex-wrap items-center gap-x-2 gap-y-1 rounded-md border border-blue-200 bg-blue-50/50 px-3 py-1.5 text-sm dark:border-blue-900 dark:bg-blue-950/30">
      <Info className="size-4 shrink-0 text-blue-600 dark:text-blue-400" />
      <span>
        {countLabel} {isSingular ? 'run' : 'runs'} started before this window{' '}
        {isSingular ? 'is' : 'are'} still active.
      </span>
      <Button
        variant="link"
        size="xs"
        className="h-auto p-0"
        onClick={showActiveRuns}
      >
        View all active runs
      </Button>
    </div>
  );
}
