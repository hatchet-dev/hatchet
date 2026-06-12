import { useSidebar } from '@/components/hooks/use-sidebar';
import { useTheme } from '@/components/hooks/use-theme';
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';
import { useBreadcrumbs } from '@/hooks/use-breadcrumbs';
import useCloud from '@/hooks/use-cloud';
import { useTenantDetails } from '@/hooks/use-tenant';
import { useTenantHomeRoute } from '@/hooks/use-tenant-home-route';
import { TenantMember, User } from '@/lib/api';
import { cn } from '@/lib/utils';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import {
  Link,
  useMatchRoute,
  useNavigate,
  useParams,
} from '@tanstack/react-router';
import {
  Check,
  ChevronDown,
  LogOut,
  Menu,
  Monitor,
  Moon,
  Sun,
  UserCircle2,
} from 'lucide-react';
import React from 'react';

interface TopNavProps {
  user?: User;
  tenantMemberships: TenantMember[];
}

const THEME_OPTIONS = [
  { value: 'light' as const, label: 'Light', icon: Sun },
  { value: 'dark' as const, label: 'Dark', icon: Moon },
  { value: 'system' as const, label: 'System', icon: Monitor },
];

function AccountDropdown({ user }: { user?: User }) {
  const [open, setOpen] = React.useState(false);
  const { logoutMutation } = useUserUniverse();
  const { theme, setTheme } = useTheme();

  if (!user) {
    return null;
  }

  const displayName = user.name || user.email;
  const activeThemeOption =
    THEME_OPTIONS.find((option) => option.value === theme) ?? THEME_OPTIONS[2];
  const ThemeIcon = activeThemeOption.icon;

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          aria-label="Open account menu"
          className={cn(
            'justify-between gap-2 bg-muted/20 shadow-none hover:bg-muted/30',
            open && 'bg-muted/30',
          )}
        >
          <UserCircle2 className="size-4" />
          <span className="max-w-32 truncate text-sm font-medium">
            {displayName}
          </span>
          <ChevronDown className="size-3.5 text-muted-foreground" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-64">
        <div className="px-2 py-1.5">
          <p className="truncate text-sm font-semibold">
            {user.name || user.email}
          </p>
          <p className="truncate text-xs text-muted-foreground">{user.email}</p>
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuSub>
          <DropdownMenuSubTrigger className="cursor-pointer">
            <ThemeIcon className="mr-2 size-4" />
            Theme: {activeThemeOption.label}
          </DropdownMenuSubTrigger>
          <DropdownMenuSubContent>
            {THEME_OPTIONS.map((option) => (
              <DropdownMenuItem
                key={option.value}
                variant="interactive"
                className="cursor-pointer"
                onClick={() => setTheme(option.value)}
              >
                <option.icon className="mr-2 size-4" />
                {option.label}
                {theme === option.value && (
                  <Check className="ml-auto size-4" />
                )}
              </DropdownMenuItem>
            ))}
          </DropdownMenuSubContent>
        </DropdownMenuSub>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          variant="interactive"
          className="cursor-pointer"
          onClick={() => logoutMutation.mutate()}
          disabled={logoutMutation.isPending}
        >
          <LogOut className="mr-2 size-4" />
          {logoutMutation.isPending ? 'Logging out...' : 'Log out'}
          <DropdownMenuShortcut>⇧⌘Q</DropdownMenuShortcut>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
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
          <Notifications />
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
          <Notifications />
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
