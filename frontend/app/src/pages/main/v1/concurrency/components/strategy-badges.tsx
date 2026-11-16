import {
  limitStrategyClassName,
  scopeClassName,
  scopeLabel,
} from './concurrency-format';
import { V1ConcurrencyStrategy } from '@/lib/api';
import { cn } from '@/lib/utils';

type StrategyBadgeProps = { strategy: V1ConcurrencyStrategy };

export function ScopeBadge({ strategy }: StrategyBadgeProps) {
  return (
    <span
      className={cn(
        'shrink-0 rounded-full border px-1.5 text-[10.5px] font-medium leading-4',
        scopeClassName[strategy.scope],
      )}
    >
      {scopeLabel[strategy.scope]}
    </span>
  );
}

export function LimitStrategyBadge({ strategy }: StrategyBadgeProps) {
  return (
    <span
      className={cn(
        'inline-block max-w-full truncate rounded border px-1.5 font-mono text-[10.5px]',
        limitStrategyClassName[strategy.limitStrategy],
      )}
    >
      {strategy.limitStrategy}
    </span>
  );
}
