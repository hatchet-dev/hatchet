import {
  V1_COLLAPSED_SIDEBAR_WIDTH,
  V1_DEFAULT_EXPANDED_SIDEBAR_WIDTH,
  V1_SIDEBAR_COLLAPSED_KEY,
  V1_SIDEBAR_WIDTH_EXPANDED_KEY,
  V1_SIDEBAR_WIDTH_LEGACY_KEY,
} from '@/components/layout/nav-constants';
import { useSidebar } from '@/components/sidebar-provider';
import { useTheme } from '@/components/theme-provider';
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
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { usePendingInvites } from '@/hooks/use-pending-invites';
import { useTenantDetails } from '@/hooks/use-tenant';
import api, { TenantMember, User } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { cn } from '@/lib/utils';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { VersionInfo } from '@/pages/main/info/components/version-info';
import { appRoutes } from '@/router';
import { useMutation } from '@tanstack/react-query';
import { useMatchRoute, useNavigate, useParams } from '@tanstack/react-router';
import { Menu } from 'lucide-react';
import React from 'react';
import { useEffect, useMemo, useState } from 'react';
import {
  BiBook,
  BiCalendar,
  BiChat,
  BiHelpCircle,
  BiLogoDiscordAlt,
  BiSolidGraduation,
  BiUserCircle,
  BiEnvelope,
} from 'react-icons/bi';

function HelpDropdown() {
  const meta = useApiMeta();
  const navigate = useNavigate();
  const { tenant } = useTenantDetails();

  const hasPylon = useMemo(() => {
    if (!meta.data?.pylonAppId) {
      return null;
    }

    return !!meta.data.pylonAppId;
  }, [meta]);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="icon" aria-label="Help Menu">
          <BiHelpCircle className="h-6 w-6 cursor-pointer text-foreground" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-56" align="end" forceMount>
        {hasPylon && (
          <DropdownMenuItem onClick={() => (window as any).Pylon('show')}>
            <BiChat className="mr-2" />
            Chat with Support
          </DropdownMenuItem>
        )}
        <DropdownMenuItem
          onClick={() => window.open('https://docs.hatchet.run', '_blank')}
        >
          <BiBook className="mr-2" />
          Documentation
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() =>
            window.open('https://discord.com/invite/ZMeUafwH89', '_blank')
          }
        >
          <BiLogoDiscordAlt className="mr-2" />
          Join Discord
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() =>
            window.open('https://hatchet.run/office-hours', '_blank')
          }
        >
          <BiCalendar className="mr-2" />
          Schedule Office Hours
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => {
            if (!tenant) {
              return;
            }

            navigate({
              to: appRoutes.tenantOnboardingGetStartedRoute.to,
              params: { tenant: tenant.metadata.id },
            });
          }}
        >
          <BiSolidGraduation className="mr-2" />
          Restart Tutorial
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function AccountDropdown({ user }: { user: User }) {
  const navigate = useNavigate();

  const { handleApiError } = useApiError({});

  const { toggleTheme } = useTheme();

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
        <Button variant="icon" aria-label="User Menu">
          <BiUserCircle className="h-6 w-6 cursor-pointer text-foreground" />
          {(pendingInvitesQuery.data ?? 0) > 0 && (
            <div className="absolute -right-0.5 -top-0.5 h-2.5 w-2.5 animate-pulse rounded-full border-2 border-background bg-blue-500"></div>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-56" align="end" forceMount>
        <DropdownMenuLabel className="font-normal">
          <div className="flex flex-col space-y-1">
            <p className="text-sm font-medium leading-none" data-cy="user-name">
              {user.name || user.email}
            </p>
            <p className="text-xs leading-none text-gray-700 dark:text-gray-300">
              {user.email}
            </p>
          </div>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {(pendingInvitesQuery.data ?? 0) > 0 && (
          <>
            <DropdownMenuItem
              onClick={() =>
                navigate({ to: appRoutes.onboardingInvitesRoute.to })
              }
            >
              <BiEnvelope className="mr-2" />
              Invites ({pendingInvitesQuery.data})
            </DropdownMenuItem>
            <DropdownMenuSeparator />
          </>
        )}
        <DropdownMenuItem>
          <VersionInfo />
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => toggleTheme()}>
          Toggle Theme
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => logoutMutation.mutate()}>
          Log out
          <DropdownMenuShortcut>⇧⌘Q</DropdownMenuShortcut>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

interface MainNavProps {
  user: User;
  tenantMemberships: TenantMember[];
}

export default function MainNav({ user }: MainNavProps) {
  const { toggleSidebarOpen } = useSidebar();
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

  // Keep the header aligned with the v1 sidebar column by mirroring its width.
  // This only affects md+ layouts; mobile uses the standard header layout.
  const defaultExpandedWidth = (() => {
    if (typeof window === 'undefined') {
      return V1_DEFAULT_EXPANDED_SIDEBAR_WIDTH;
    }

    try {
      // Back-compat: previous implementation stored this under `v1SidebarWidth`.
      const legacy = window.localStorage.getItem(V1_SIDEBAR_WIDTH_LEGACY_KEY);
      return legacy ? JSON.parse(legacy) : V1_DEFAULT_EXPANDED_SIDEBAR_WIDTH;
    } catch {
      return V1_DEFAULT_EXPANDED_SIDEBAR_WIDTH;
    }
  })();

  const [storedExpandedWidth] = useLocalStorageState(
    V1_SIDEBAR_WIDTH_EXPANDED_KEY,
    defaultExpandedWidth,
  );
  const [storedCollapsed] = useLocalStorageState(
    V1_SIDEBAR_COLLAPSED_KEY,
    false,
  );
  const [isWide, setIsWide] = useState(() =>
    typeof window !== 'undefined' ? window.innerWidth >= 768 : false,
  );

  useEffect(() => {
    const handleResize = () => setIsWide(window.innerWidth >= 768);
    handleResize();
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  const headerSidebarWidth = useMemo(() => {
    if (!isWide) {
      return undefined;
    }
    return storedCollapsed ? V1_COLLAPSED_SIDEBAR_WIDTH : storedExpandedWidth;
  }, [isWide, storedCollapsed, storedExpandedWidth]);

  return (
    <header className="z-50 h-16 w-full border-b bg-background">
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

        <div className="ml-auto flex items-center gap-2">
          <HelpDropdown />
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
              onClick={onLogoClick}
              aria-label="Go to Runs"
              className="rounded-sm outline-none ring-offset-background focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            >
              <HatchetLogo variant="mark" className="h-5 w-5" />
            </button>
          ) : (
            <button
              type="button"
              onClick={onLogoClick}
              aria-label="Go to Runs"
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
          <HelpDropdown />
          <AccountDropdown user={user} />
        </div>
      </div>
    </header>
  );
}
