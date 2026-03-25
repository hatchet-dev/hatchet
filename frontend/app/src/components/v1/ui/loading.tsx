import { Icons } from '@/components/v1/ui/icons.tsx';
import { cn } from '@/lib/utils';

export function Spinner({ className }: { className?: string }) {
  return (
    <Icons.spinner className={cn('mr-2 h-4 w-4 animate-spin', className)} />
  );
}

export function Loading({ className }: { className?: string }) {
  return (
    <div className={cn('flex h-full w-full flex-1 flex-row', className)}>
      <Spinner />
    </div>
  );
}
