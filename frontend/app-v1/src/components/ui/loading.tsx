import { Icons } from '@/components/ui/icons.tsx';
import { cn } from '@/lib/utils';

export function Spinner() {
  return <Icons.spinner className="mr-2 h-4 w-4 animate-spin" />;
}

export function Loading({ className }: { className?: string }) {
  return (
    <div className={cn('flex flex-row flex-1 w-full h-full', className)}>
      <Spinner />
    </div>
  );
}
