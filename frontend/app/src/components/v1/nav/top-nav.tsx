import { useSidebar } from '@/components/hooks/use-sidebar';
import { OrganizationSelector } from '@/components/v1/molecules/nav-bar/organization-selector';
import { TenantSwitcher } from '@/components/v1/molecules/nav-bar/tenant-switcher';
import { Notifications } from '@/components/v1/nav/notifications';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/v1/ui/breadcrumb';
import { Button } from '@/components/v1/ui/button';
import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';
import { useBreadcrumbs } from '@/hooks/use-breadcrumbs';
import { useTenantDetails } from '@/hooks/use-tenant';
import { useTenantHomeRoute } from '@/hooks/use-tenant-home-route';
import { TenantMember, User } from '@/lib/api';
import { cn } from '@/lib/utils';
import useCloud from '@/pages/auth/hooks/use-cloud';
import { appRoutes } from '@/router';
import {
  Link,
  useMatchRoute,
  useNavigate,
  useParams,
} from '@tanstack/react-router';
import { Menu } from 'lucide-react';
import React from 'react';

interface TopNavProps {
  user?: User;
  tenantMemberships: TenantMember[];
}

export default function TopNav({ user, tenantMemberships }: TopNavProps) {
  const {
    toggleSidebarOpen,
    isWide,
    sidebarWidth: headerSidebarWidth,
    collapsed: storedCollapsed,
    setCollapsed: setStoredCollapsed,
  } = useSidebar();
  const breadcrumbs = useBreadcrumbs();
  const navigate = useNavigate();
  const matchRoute = useMatchRoute();
  const params = useParams({ strict: false }) as { tenant?: string };
  const tenantParamInPath = params.tenant;
  const { homeRoute } = useTenantHomeRoute(tenantParamInPath);

  const isOnTenantRoute = Boolean(
    matchRoute({
      to: appRoutes.tenantRoute.to,
      params: tenantParamInPath ? { tenant: tenantParamInPath } : undefined,
      fuzzy: true,
    }),
  );

  const onLogoClick = () => {
    if (isOnTenantRoute && tenantParamInPath) {
      navigate({
        to: homeRoute,
        params: { tenant: tenantParamInPath },
      });
      return;
    }

    navigate({ to: appRoutes.authenticatedRoute.to });
  };

  const onDesktopLogoClick = () => {
    // On desktop, clicking the logo acts as a collapse/expand affordance.
    // Mobile keeps the navigation behavior (see mobile header).
    if (!isWide) {
      onLogoClick();
      return;
    }

    setStoredCollapsed(!storedCollapsed);
  };

  const { isCloudEnabled } = useCloud();
  const { tenant } = useTenantDetails();
  const showTenantSwitcher =
    !!user && tenantMemberships?.length > 0 && !!tenant;

  return (
    <header className="z-50 h-16 w-full  bg-background">
      {/* Mobile header */}
      <div className="flex h-16 items-center px-4 md:hidden">
        <div className="flex items-center gap-3">
          <Button
            variant="icon"
            onClick={() => toggleSidebarOpen()}
            aria-label="Toggle sidebar"
            size="icon"
          >
            <Menu className="size-4" />
          </Button>
          <button
            type="button"
            onClick={onLogoClick}
            aria-label="Go to Runs"
            className="rounded-sm outline-none ring-offset-background focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          >
            <HatchetLogo variant="mark" className="h-5 w-5" />
          </button>
        </div>

        <div className="flex ml-auto items-center justify-end gap-2">
          {showTenantSwitcher &&
            (isCloudEnabled ? (
              <OrganizationSelector memberships={tenantMemberships} />
            ) : (
              <TenantSwitcher memberships={tenantMemberships} />
            ))}
          <Notifications />
        </div>
      </div>

      {/* Desktop header (aligned to v1 sidebar column) */}
      <div
        className="hidden h-16 md:grid md:grid-cols-[var(--v1-sidebar-width)_minmax(0,1fr)_auto] md:items-center"
        style={
          headerSidebarWidth
            ? ({
                ['--v1-sidebar-width' as any]: `${headerSidebarWidth}px`,
              } as React.CSSProperties)
            : undefined
        }
      >
        <div
          className={cn(
            'flex h-16 items-center',
            // Match the icon position in the expanded sidebar (px-4 container + pl-2 button => 24px).
            // In collapsed mode, center within the column to match the icon-only sidebar.
            storedCollapsed ? 'justify-center' : 'pl-6',
          )}
        >
          {storedCollapsed ? (
            <button
              type="button"
              onClick={onDesktopLogoClick}
              aria-label="Expand sidebar"
              className="rounded-sm outline-none ring-offset-background focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            >
              <HatchetLogo variant="mark" className="h-5 w-5" />
            </button>
          ) : (
            <button
              type="button"
              onClick={onDesktopLogoClick}
              aria-label="Collapse sidebar"
              className="flex items-center gap-2 rounded-sm outline-none ring-offset-background focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            >
              <HatchetLogo variant="mark" className="h-4 w-4" />
              <HatchetLogo variant="wordmark" className="h-4 w-auto" />
            </button>
          )}
        </div>

        <div className="min-w-0 px-8">
          {breadcrumbs.length > 0 && (
            <Breadcrumb>
              <BreadcrumbList>
                {breadcrumbs.map((crumb, index) => (
                  <React.Fragment key={index}>
                    {index > 0 && <BreadcrumbSeparator />}
                    <BreadcrumbItem>
                      {crumb.isCurrentPage ? (
                        <BreadcrumbPage>{crumb.label}</BreadcrumbPage>
                      ) : (
                        <BreadcrumbLink asChild>
                          <Link to={crumb.href}>{crumb.label}</Link>
                        </BreadcrumbLink>
                      )}
                    </BreadcrumbItem>
                  </React.Fragment>
                ))}
              </BreadcrumbList>
            </Breadcrumb>
          )}
        </div>

        <div className="flex items-center justify-end gap-2 pr-4">
          {showTenantSwitcher &&
            (isCloudEnabled ? (
              <OrganizationSelector memberships={tenantMemberships} />
            ) : (
              <TenantSwitcher memberships={tenantMemberships} />
            ))}
          <Notifications />
        </div>
      </div>
    </header>
  );
}
