import { ReactNode } from 'react';
interface SheetViewLayoutProps {
  children: ReactNode;
  sheet: ReactNode;
}

export function SheetViewLayout({ children, sheet }: SheetViewLayoutProps) {
  return (
    <div className="flex h-full w-full flex-row">
      <div className="px-8 py-12 overflow-y-scroll flex-grow">
        {children}
      </div>
      {sheet}
    </div>
  );
}
