import { Icons } from '@/components/ui/icons.tsx';
import { cn } from '@/lib/utils';

export function Spinner({ className }: { className?: string }) {
  return (
    <div className={cn('flex flex-row flex-1 w-full h-full', className)}>
      <Icons.spinner className="mr-2 h-4 w-4 animate-spin" />
    </div>
  );
}
