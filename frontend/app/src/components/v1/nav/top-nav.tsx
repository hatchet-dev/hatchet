import { useSidebar } from '@/components/hooks/use-sidebar';
import { useTheme } from '@/components/hooks/use-theme';
import { OrganizationSelector } from '@/components/v1/molecules/nav-bar/organization-selector';
import { TenantSwitcher } from '@/components/v1/molecules/nav-bar/tenant-switcher';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/v1/ui/breadcrumb';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';
import { useBreadcrumbs } from '@/hooks/use-breadcrumbs';
import { usePendingInvites } from '@/hooks/use-pending-invites';
import { useTenantDetails } from '@/hooks/use-tenant';
import api, { TenantMember, User } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { cn } from '@/lib/utils';
import useCloud from '@/pages/auth/hooks/use-cloud';
import { appRoutes } from '@/router';
import { useMutation } from '@tanstack/react-query';
import { useMatchRoute, useNavigate, useParams } from '@tanstack/react-router';
import { ChevronDown, Menu } from 'lucide-react';
import React from 'react';
import {
  BiEnvelope,
  BiLogOut,
  BiMoon,
  BiSun,
  BiUserCircle,
} from 'react-icons/bi';

function AccountDropdown({ user }: { user?: User }) {
  const navigate = useNavigate();

  const { handleApiError } = useApiError({});

  const { toggleTheme, theme } = useTheme();

  // Check for pending invites to show the Invites menu item
  const { pendingInvitesQuery } = usePendingInvites();

  const logoutMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: async () => {
      await api.userUpdateLogout();
    },
    onSuccess: () => {
      navigate({ to: appRoutes.authLoginRoute.to });
    },
    onError: handleApiError,
  });

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          aria-label="User Menu"
          className="relative max-w-[220px] justify-between gap-2 bg-muted/20 px-2 shadow-none hover:bg-muted/30"
          disabled={!user}
        >
          <div className="flex min-w-0 flex-1 items-center gap-2 text-left">
            <BiUserCircle className="size-5 shrink-0 text-foreground" />
            <span
              className="hidden min-w-0 flex-1 truncate lg:block"
              data-cy="user-name"
            >
              {user?.name || user?.email}
              {/* TODO-DESIGN: add a loading state */}
            </span>
          </div>
          <ChevronDown className="size-4 shrink-0 opacity-60" />
          {(pendingInvitesQuery.data ?? 0) > 0 && (
            <div className="absolute -right-0.5 -top-0.5 h-2.5 w-2.5 animate-pulse rounded-full border-2 border-background bg-blue-500"></div>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-56" align="end" forceMount>
        <DropdownMenuLabel className="font-normal">
          <div className="flex flex-col space-y-1">
            <p className="text-sm font-medium leading-none" data-cy="user-name">
              {user?.name || user?.email}
            </p>
            <p className="text-xs leading-none text-gray-700 dark:text-gray-300">
              {user?.email}
            </p>
          </div>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {(pendingInvitesQuery.data ?? 0) > 0 && (
          <>
            <DropdownMenuItem
              variant="interactive"
              onClick={() =>
                navigate({ to: appRoutes.onboardingInvitesRoute.to })
              }
            >
              <BiEnvelope className="mr-2 size-4" />
              Invites ({pendingInvitesQuery.data})
            </DropdownMenuItem>
            <DropdownMenuSeparator />
          </>
        )}
        <DropdownMenuItem variant="interactive" onClick={() => toggleTheme()}>
          {theme === 'dark' ? (
            <BiSun className="mr-2 size-4" />
          ) : (
            <BiMoon className="mr-2 size-4" />
          )}
          Theme: {theme === 'dark' ? 'Dark' : 'Light'}
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          variant="interactive"
          onClick={() => logoutMutation.mutate()}
        >
          <BiLogOut className="mr-2 size-4" />
          Log out
          <DropdownMenuShortcut>⇧⌘Q</DropdownMenuShortcut>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

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
        to: appRoutes.tenantRunsRoute.to,
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
          <AccountDropdown user={user} />
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
                        <BreadcrumbLink href={crumb.href}>
                          {crumb.label}
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
          <AccountDropdown user={user} />
        </div>
      </div>
    </header>
  );
}
