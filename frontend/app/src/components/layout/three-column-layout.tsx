import { cn } from '@/lib/utils';
import { ReactNode } from 'react';

type ThreeColumnLayoutProps = {
  sidebar: ReactNode;
  children: ReactNode;
  sidePanel?: ReactNode;
  className?: string;
  mainClassName?: string;
  /** Enables container queries on the main column (used by v1) */
  mainContainerType?: 'inline-size' | 'normal';
};

export function ThreeColumnLayout({
  sidebar,
  children,
  sidePanel,
  className,
  mainClassName,
  mainContainerType = 'normal',
}: ThreeColumnLayoutProps) {
  return (
    <div
      className={cn(
        'relative grid h-full w-full min-h-0 min-w-0 grid-rows-1 grid-cols-[0px_1fr_auto] overflow-hidden md:grid-cols-[auto_1fr_auto]',
        className,
      )}
    >
      <div className="col-start-1 row-start-1 min-h-0">{sidebar}</div>

      <div
        className={cn('col-start-2 row-start-1 min-h-0 min-w-0', mainClassName)}
        style={
          mainContainerType === 'inline-size'
            ? { containerType: 'inline-size' }
            : undefined
        }
      >
        {children}
      </div>

      <div className="col-start-3 row-start-1 min-h-0">{sidePanel}</div>
    </div>
  );
}
