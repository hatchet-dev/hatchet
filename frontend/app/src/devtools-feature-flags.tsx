import { Button } from '@/components/v1/ui/button';
import {
  FeatureFlagId,
  useFeatureFlagOverrides,
} from '@/hooks/use-feature-flags';
import { cn } from '@/lib/utils';
import { useAppContext } from '@/providers/app-context';
import { useQueryClient } from '@tanstack/react-query';
import { useCallback, useSyncExternalStore } from 'react';

type OverrideChoice = 'default' | 'on' | 'off';

const CHOICES: OverrideChoice[] = ['default', 'on', 'off'];

// The panel doesn't know each call site's `isEnabledIfNoPosthog`, so it reads
// whatever the app last evaluated from the query cache instead of re-querying.
// The snapshot is a string so useSyncExternalStore sees a stable value between
// unrelated cache updates.
function useEvaluatedFlags(tenantId: string | undefined) {
  const queryCache = useQueryClient().getQueryCache();

  const subscribe = useCallback(
    (onChange: () => void) => queryCache.subscribe(onChange),
    [queryCache],
  );

  const snapshot = useSyncExternalStore(subscribe, () => {
    const values: Partial<Record<FeatureFlagId, boolean>> = {};
    const queries = queryCache
      .findAll({ queryKey: ['feature-flag', tenantId] })
      .sort((a, b) => a.state.dataUpdatedAt - b.state.dataUpdatedAt);
    for (const query of queries) {
      const flag = query.queryKey[2] as FeatureFlagId;
      const data = query.state.data as { isEnabled?: boolean } | undefined;
      if (data?.isEnabled !== undefined) {
        values[flag] = data.isEnabled;
      } else if (query.state.status === 'error') {
        // Mirrors useIsFeatureEnabled, which falls back to isEnabledIfNoPosthog on failure.
        values[flag] = query.queryKey[3] as boolean;
      }
    }
    return JSON.stringify(values);
  });

  return JSON.parse(snapshot) as Partial<Record<FeatureFlagId, boolean>>;
}

export function FeatureFlagsPanel() {
  const { tenantId } = useAppContext();
  const [overrides, setOverrides] = useFeatureFlagOverrides();
  const evaluated = useEvaluatedFlags(tenantId);

  const setChoice = (flag: FeatureFlagId, choice: OverrideChoice) => {
    setOverrides((prev) => {
      const next = { ...prev };
      if (choice === 'default') {
        delete next[flag];
      } else {
        next[flag] = choice === 'on';
      }
      return next;
    });
  };

  return (
    <div className="flex flex-col gap-3 p-4 text-sm">
      <div className="flex items-center justify-between">
        <span className="font-medium">Feature flags</span>
        <Button
          size="sm"
          variant="ghost"
          disabled={Object.keys(overrides).length === 0}
          onClick={() => setOverrides({})}
        >
          Reset all
        </Button>
      </div>
      <div className="flex flex-col gap-2">
        {Object.values(FeatureFlagId).map((flag) => {
          const override = overrides[flag];
          const current: OverrideChoice =
            override === undefined ? 'default' : override ? 'on' : 'off';
          const effective = override ?? evaluated[flag];

          return (
            <div key={flag} className="flex items-center gap-3">
              <code className="min-w-0 flex-1 truncate">{flag}</code>
              <span
                className={cn(
                  'w-8 text-xs',
                  effective === undefined && 'text-muted-foreground',
                )}
                title={
                  effective === undefined
                    ? 'Not evaluated on this page yet'
                    : undefined
                }
              >
                {effective === undefined ? '—' : effective ? 'on' : 'off'}
              </span>
              <div className="flex gap-1">
                {CHOICES.map((choice) => (
                  <Button
                    key={choice}
                    size="sm"
                    variant={current === choice ? 'secondary' : 'outline'}
                    onClick={() => setChoice(flag, choice)}
                    className="capitalize"
                  >
                    {choice}
                  </Button>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
