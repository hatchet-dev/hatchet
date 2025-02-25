import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';

import { GrUpgrade } from 'react-icons/gr';

import {
  Link,
  useLocation,
  useNavigate,
  useSearchParams,
} from 'react-router-dom';
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
import { useMemo } from 'react';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { VersionInfo } from '@/pages/main/info/components/version-info';
import { useTenant } from '@/lib/atoms';
import { routes } from '@/router';
import { TooltipContent, TooltipProvider } from '@/components/v1/ui/tooltip';
import { Tooltip, TooltipTrigger } from '@/components/v1/ui/tooltip';

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
          onClick={() =>
            window.open('https://docs.hatchet.run/home/basics/steps', '_blank')
          }
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

function VersionUpgradeButton() {
  const { tenant } = useTenant();
  const tenantVersion = tenant?.version || TenantVersion.V0;
  const { pathname } = useLocation();
  const [params] = useSearchParams();

  const versionedRoutes = useMemo(
    () =>
      routes
        .at(0)
        ?.children?.find((r) => r.path === '/v1/')
        ?.children?.find((r) => r.path === '/v1/' && r.children?.length)
        ?.children?.map((c) => c.path)
        ?.map((p) => p?.replace('/v1', '')) || [],
    [],
  );

  const shouldShowVersionUpgradeButton =
    versionedRoutes.includes(pathname) && // It is a versioned route
    !pathname.includes('/v1') && // The user is not already on the v1 version
    tenantVersion === TenantVersion.V1; // The tenant is on the v1 version

  if (!shouldShowVersionUpgradeButton) {
    return null;
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>
          {' '}
          <Link
            to={{
              pathname: '/v1' + pathname,
              search: params.toString(),
            }}
          >
            <Button
              variant="ghost"
              className="relative h-10 w-10 rounded-full p-1 animate-bounce"
              aria-label="Upgrade version"
            >
              <GrUpgrade className="h-6 w-6 text-foreground cursor-pointer text-red-500" />
            </Button>
          </Link>{' '}
        </TooltipTrigger>
        <TooltipContent>Switch to v1</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

interface MainNavProps {
  user: User;
}

export default function MainNav({ user }: MainNavProps) {
  const { toggleSidebarOpen } = useSidebar();
  const { theme } = useTheme();

  return (
    <div className="fixed top-0 w-screen h-16 border-b">
      <div className="flex h-16 items-center pr-4 pl-4">
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
        <div className="ml-auto flex items-center">
          <HelpDropdown />
          <AccountDropdown user={user} />
          <VersionUpgradeButton />
        </div>
      </div>
    </div>
  );
}
