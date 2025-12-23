import { cn } from '@/lib/utils';
import { ReactNode } from 'react';

type AppLayoutProps = {
  header: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
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
  footer,
  contentScroll = true,
  className,
}: AppLayoutProps) {
  const hasFooter = Boolean(footer);
  return (
    <div
      className={cn(
        'grid h-full w-full min-h-0 min-w-0 overflow-hidden',
        hasFooter
          ? 'grid-rows-[64px_minmax(0,1fr)_auto]'
          : 'grid-rows-[64px_minmax(0,1fr)]',
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

      {hasFooter ? (
        <div className="min-w-0 border-t bg-background">
          <div className="mx-auto flex min-h-9 w-full min-w-0 items-center px-2">
            {footer}
          </div>
        </div>
      ) : null}
    </div>
  );
}
