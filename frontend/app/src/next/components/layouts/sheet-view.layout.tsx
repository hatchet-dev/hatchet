import { ReactNode } from 'react';

interface SheetViewLayoutProps {
  children: ReactNode;
  sheet: ReactNode;
}

export function SheetViewLayout({ children, sheet }: SheetViewLayoutProps) {
  return (
    <div className="flex h-full w-full flex-row gap-4">
      <div className="overflow-y-auto flex-grow p-4 py-8">{children}</div>
      {sheet}
    </div>
  );
}
