import { Button } from '@/components/v1/ui/button';
import { cn } from '@/lib/utils';
import { ReactQueryDevtoolsPanel } from '@tanstack/react-query-devtools';
import { useRouter } from '@tanstack/react-router';
import { TanStackRouterDevtoolsPanel } from '@tanstack/react-router-devtools';
import { useState } from 'react';

export default function DevtoolsFooter() {
  const router = useRouter();
  const [openQuery, setOpenQuery] = useState(false);
  const [openRouter, setOpenRouter] = useState(false);

  const anyOpen = openQuery || openRouter;
  return (
    <div className="flex w-full min-w-0 items-center justify-end">
      <div className="relative flex items-center gap-2">
        <Button
          size="sm"
          variant={openQuery ? 'secondary' : 'outline'}
          onClick={() => setOpenQuery((v) => !v)}
        >
          Query
        </Button>
        <Button
          size="sm"
          variant={openRouter ? 'secondary' : 'outline'}
          onClick={() => setOpenRouter((v) => !v)}
        >
          Router
        </Button>

        {anyOpen ? (
          <Button
            size="sm"
            variant="ghost"
            onClick={() => {
              setOpenQuery(false);
              setOpenRouter(false);
            }}
          >
            Close
          </Button>
        ) : null}

        {anyOpen ? (
          <div
            className={cn(
              'absolute bottom-full right-0 mb-2 overflow-hidden rounded-md border bg-background shadow-lg',
              'w-[min(820px,calc(100vw-16px))] max-h-[70vh]',
            )}
          >
            <div className="min-h-0 overflow-auto">
              {openQuery ? <ReactQueryDevtoolsPanel /> : null}
              {openRouter ? (
                <TanStackRouterDevtoolsPanel router={router} />
              ) : null}
            </div>
          </div>
        ) : null}
      </div>
    </div>
  );
}
