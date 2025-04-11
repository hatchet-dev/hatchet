import { NavItem } from '@/next/pages/authenticated/dashboard/components/sidebar/main-nav';
import * as React from 'react';

export interface BreadcrumbData {
  title: React.ReactNode;
  url: string;
  siblings?: NavItem[];
  section?: string;
  icon?: React.ElementType;
  alwaysShowTitle?: boolean;
  alwaysShowIcon?: boolean;
}

interface BreadcrumbContextType {
  breadcrumbs: BreadcrumbData[];
  setBreadcrumbs: (breadcrumbs: BreadcrumbData[]) => void;
}

const BreadcrumbContext = React.createContext<
  BreadcrumbContextType | undefined
>(undefined);

export function BreadcrumbProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [breadcrumbs, setBreadcrumbs] = React.useState<BreadcrumbData[]>([]);

  return (
    <BreadcrumbContext.Provider
      value={{
        breadcrumbs,
        setBreadcrumbs,
      }}
    >
      {children}
    </BreadcrumbContext.Provider>
  );
}

export function useBreadcrumbs(): BreadcrumbContextType {
  const context = React.useContext(BreadcrumbContext);

  if (context === undefined) {
    throw new Error('useBreadcrumbs must be used within a BreadcrumbProvider');
  }

  return context;
}
