import { Loader2 } from 'lucide-react';

export function SsoLoadingState() {
  return (
    <div className="mx-auto max-w-3xl">
      <div className="flex h-32 items-center justify-center text-muted-foreground">
        <Loader2 className="mr-2 h-4 w-4 animate-spin" /> Loading…
      </div>
    </div>
  );
}
