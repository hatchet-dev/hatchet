import { cn } from '@/lib/utils';
import { ReactNode } from 'react';

type AppLayoutProps = {
  header: ReactNode;
  children: ReactNode;
  /**
   * When true, the content area becomes the scroll container.
   * When false, content is overflow-hidden (useful when a child layout owns scrolling).
   */
  contentScroll?: boolean;
  className?: string;
};

export function AppLayout({
  header,
  children,
  contentScroll = true,
  className,
}: AppLayoutProps) {
  return (
    <div
      className={cn(
        'grid h-full w-full min-h-0 min-w-0 grid-rows-[64px_minmax(0,1fr)] overflow-hidden',
        className,
      )}
    >
      {header}

      <div className="min-h-0 min-w-0 overflow-hidden">
        <div
          className={cn(
            'h-full w-full min-h-0 min-w-0',
            contentScroll ? 'overflow-auto' : 'overflow-hidden',
          )}
        >
          {children}
        </div>
      </div>
    </div>
  );
}


