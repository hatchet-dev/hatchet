import React from 'react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

import { useLocation, useNavigate } from 'react-router-dom';
import api, { TenantVersion, User } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import hatchet from '@/assets/hatchet_logo.png';
import hatchetDark from '@/assets/hatchet_logo_dark.png';
import { useSidebar } from '@/components/sidebar-provider';
import {
  BiBook,
  BiCalendar,
  BiChat,
  BiHelpCircle,
  BiLogoDiscordAlt,
  BiSolidGraduation,
  BiUserCircle,
} from 'react-icons/bi';
import { useTheme } from '@/components/theme-provider';
import { useEffect, useMemo } from 'react';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { VersionInfo } from '@/pages/main/info/components/version-info';
import { useTenant } from '@/lib/atoms';
import { routes } from '@/router';
import { Banner, BannerProps } from './banner';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/v1/ui/breadcrumb';
import { useBreadcrumbs } from '@/hooks/use-breadcrumbs';

function HelpDropdown() {
  const meta = useApiMeta();

  const hasPylon = useMemo(() => {
    if (!meta.data?.pylonAppId) {
      return null;
    }

    return !!meta.data.pylonAppId;
  }, [meta]);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          className="relative h-10 w-10 rounded-full p-1"
          aria-label="Help Menu"
        >
          <BiHelpCircle className="h-6 w-6 text-foreground cursor-pointer" />
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
          onClick={() => window.open('/onboarding/get-started', '_self')}
        >
          <BiSolidGraduation className="mr-2" />
          Restart Tutorial
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function AccountDropdown({ user }: MainNavProps) {
  const navigate = useNavigate();
  const { tenant } = useTenant();

  const { handleApiError } = useApiError({});

  const { toggleTheme } = useTheme();

  const logoutMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: async () => {
      await api.userUpdateLogout();
    },
    onSuccess: () => {
      navigate('/auth/login');
    },
    onError: handleApiError,
  });

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          className="relative h-10 w-10 rounded-full p-1"
          aria-label="User Menu"
        >
          <BiUserCircle className="h-6 w-6 text-foreground cursor-pointer" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-56" align="end" forceMount>
        <DropdownMenuLabel className="font-normal">
          <div className="flex flex-col space-y-1">
            <p className="text-sm font-medium leading-none">
              {user.name || user.email}
            </p>
            <p className="text-xs leading-none text-gray-700 dark:text-gray-300">
              {user.email}
            </p>
          </div>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem>
          <VersionInfo />
        </DropdownMenuItem>
        {tenant?.version == TenantVersion.V1 &&
          location.pathname.includes('v1') && (
            <DropdownMenuItem
              onClick={() => navigate('/workflow-runs?previewV0=true')}
            >
              View Legacy V0 Data
            </DropdownMenuItem>
          )}
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
  setHasBanner?: (state: boolean) => void;
}

export default function MainNav({ user, setHasBanner }: MainNavProps) {
  const { toggleSidebarOpen } = useSidebar();
  const { theme } = useTheme();
  const { tenant } = useTenant();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const breadcrumbs = useBreadcrumbs();

  const tenantedRoutes = useMemo(
    () =>
      routes
        .at(0)
        ?.children?.find((r) => r.path?.startsWith('/tenants/'))
        ?.children?.find(
          (r) => r.path?.startsWith('/tenants/') && r.children?.length,
        )
        ?.children?.map((c) => c.path)
        ?.map((p) => p?.replace('/tenants/:tenant', '')) || [],
    [],
  );

  const tenantVersion = tenant?.version || TenantVersion.V0;

  const banner: BannerProps | undefined = useMemo(() => {
    const pathnameWithoutTenant = pathname.replace(
      `/tenants/${tenant?.metadata.id}`,
      '',
    );

    const shouldShowVersionUpgradeButton =
      tenantedRoutes.includes(pathnameWithoutTenant) && // It is a versioned route
      !pathname.startsWith('/tenants') && // The user is not already on the v1 version
      tenantVersion === TenantVersion.V1; // The tenant is on the v1 version

    if (shouldShowVersionUpgradeButton) {
      return {
        message: (
          <>
            You are viewing legacy V0 data for a tenant that was upgraded to V1
            runtime.
          </>
        ),
        type: 'warning',
        actionText: 'View V1',
        onAction: () => {
          navigate({
            pathname: `/tenants/${tenant?.metadata.id}${pathname}`,
            search: '?previewV0=false',
          });
        },
      };
    }

    return;
  }, [navigate, pathname, tenantVersion, tenantedRoutes, tenant?.metadata.id]);

  useEffect(() => {
    if (!setHasBanner) {
      return;
    }
    setHasBanner(!!banner);
  }, [setHasBanner, banner]);

  return (
    <div className="fixed top-0 w-screen">
      {banner && <Banner {...banner} />}

      <div className="h-16 border-b">
        <div className="flex h-16 items-center pr-4 pl-4">
          <div className="flex flex-row items-center gap-x-8">
            <button
              onClick={() => toggleSidebarOpen()}
              className="flex flex-row gap-4 items-center"
            >
              <img
                src={theme == 'dark' ? hatchet : hatchetDark}
                alt="Hatchet"
                className="h-9 rounded"
              />
            </button>
            {breadcrumbs.length > 0 && (
              <Breadcrumb className="hidden md:block">
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

          <div className="ml-auto flex items-center">
            <HelpDropdown />
            <AccountDropdown user={user} />
          </div>
        </div>
      </div>
    </div>
  );
}
