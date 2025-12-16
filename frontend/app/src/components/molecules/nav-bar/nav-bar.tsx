import React from 'react';
import { ReviewedButtonTemp } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';

import { useNavigate } from 'react-router-dom';
import api, { TenantMember, User } from '@/lib/api';
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
  BiEnvelope,
} from 'react-icons/bi';
import { Menu } from 'lucide-react';
import { useTheme } from '@/components/theme-provider';
import { useMemo } from 'react';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { VersionInfo } from '@/pages/main/info/components/version-info';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/v1/ui/breadcrumb';
import { useBreadcrumbs } from '@/hooks/use-breadcrumbs';
import { usePendingInvites } from '@/hooks/use-pending-invites';
import { useTenantDetails } from '@/hooks/use-tenant';

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
        <ReviewedButtonTemp variant="icon" aria-label="Help Menu">
          <BiHelpCircle className="h-6 w-6 text-foreground cursor-pointer" />
        </ReviewedButtonTemp>
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

            navigate(`/tenants/${tenant.metadata.id}/onboarding/get-started`);
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
      navigate('/auth/login');
    },
    onError: handleApiError,
  });

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <ReviewedButtonTemp variant="icon" aria-label="User Menu">
          <BiUserCircle className="h-6 w-6 text-foreground cursor-pointer" />
          {(pendingInvitesQuery.data ?? 0) > 0 && (
            <div className="absolute -top-0.5 -right-0.5 h-2.5 w-2.5 bg-blue-500 rounded-full border-2 border-background animate-pulse"></div>
          )}
        </ReviewedButtonTemp>
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
        {(pendingInvitesQuery.data ?? 0) > 0 && (
          <>
            <DropdownMenuItem onClick={() => navigate('/onboarding/invites')}>
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
  const { theme } = useTheme();
  const breadcrumbs = useBreadcrumbs();

  return (
    <div className="fixed top-0 w-screen z-50">
      <div className="h-16 border-b bg-background">
        <div className="flex h-16 items-center pr-4 pl-4">
          <div className="flex flex-row items-center gap-x-8">
            <div className="flex items-center gap-3">
              <ReviewedButtonTemp
                variant="icon"
                onClick={() => toggleSidebarOpen()}
                aria-label="Toggle sidebar"
                size="icon"
              >
                <Menu className="size-4" />
              </ReviewedButtonTemp>
              <img
                src={theme == 'dark' ? hatchet : hatchetDark}
                alt="Hatchet"
                className="h-9 rounded"
              />
            </div>
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

          <div className="ml-auto flex items-center gap-2">
            <HelpDropdown />
            <AccountDropdown user={user} />
          </div>
        </div>
      </div>
    </div>
  );
}
