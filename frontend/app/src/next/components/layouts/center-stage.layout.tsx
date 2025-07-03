import { cn } from '@/next/lib/utils';
import { ReactNode } from 'react';

interface CenterStageLayoutProps {
  children: ReactNode;
  className?: string;
}

export function CenterStageLayout({
  children,
  className = '',
}: CenterStageLayoutProps) {
  return (
    <div
      className={cn(
        'min-h-screen w-full flex items-center justify-center',
        className,
      )}
    >
      <div className="w-full max-w-7xl px-4 sm:px-6 lg:px-8">{children}</div>
    </div>
  );
}
