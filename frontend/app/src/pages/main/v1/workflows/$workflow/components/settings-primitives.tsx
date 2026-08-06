import { ConcurrencyLimitStrategy } from '@/lib/api';

export function EmptyState({ message }: { message: string }) {
  return <p className="text-xs italic text-muted-foreground">{message}</p>;
}

export function FieldLabel({ children }: { children: React.ReactNode }) {
  return <div className="mb-1 text-xs text-muted-foreground">{children}</div>;
}

export function formatLimitStrategy(
  strategy: ConcurrencyLimitStrategy,
): string {
  switch (strategy) {
    case ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS:
      return 'Cancel In Progress';
    case ConcurrencyLimitStrategy.DROP_NEWEST:
      return 'Drop Newest';
    case ConcurrencyLimitStrategy.QUEUE_NEWEST:
      return 'Queue Newest';
    case ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN:
      return 'Group Round Robin';
    default: {
      const exhaustiveCheck: never = strategy;
      return exhaustiveCheck;
    }
  }
}
