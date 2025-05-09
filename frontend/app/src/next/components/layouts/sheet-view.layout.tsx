import { ReactNode } from 'react';
interface SheetViewLayoutProps {
  children: ReactNode;
  sheet: ReactNode;
}

export function SheetViewLayout({ children, sheet }: SheetViewLayoutProps) {
  return (
    <div className="flex h-full w-full flex-row">
      <div className="flex-1 min-w-0 h-full px-8 py-12 overflow-y-auto overflow-x-auto">
        {children}
      </div>
      {/* Note: The sheet should be responsible for its width */}
      {sheet}
    </div>
  );
}
