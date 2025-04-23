import { NavItem } from '@/next/pages/authenticated/dashboard/components/sidebar/main-nav';
import * as React from 'react';

export interface BreadcrumbData {
  title: string;
  label: React.ReactNode;
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

export function useBreadcrumbs(
  effect: () => BreadcrumbData[],
  deps: React.DependencyList,
): BreadcrumbContextType {
  const context = React.useContext(BreadcrumbContext);

  if (context === undefined) {
    throw new Error('useBreadcrumbs must be used within a BreadcrumbProvider');
  }

  React.useEffect(() => {
    const breadcrumbs = effect();
    context.setBreadcrumbs(breadcrumbs);

    return () => {
      context.setBreadcrumbs([]);
    };
  }, [effect, deps, context]);

  return context;
}
