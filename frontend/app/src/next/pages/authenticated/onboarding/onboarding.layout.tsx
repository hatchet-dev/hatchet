import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { ChevronsUpDown, Sun, Moon, LogOut } from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/next/components/ui/dropdown-menu';
import { useTheme } from '@/next/components/theme-provider';
import useUser from '@/next/hooks/use-user';
import { Button } from '@/next/components/ui/button';
import { Logo } from '@/next/components/ui/logo';
import { ROUTES } from '@/next/lib/routes';
import { UserBlock } from '../../learn/learn.layout';

export function OnboardingLayout() {
  const { toggleTheme, theme } = useTheme();
  const { logout, data: userData } = useUser();
  const location = useLocation();
  const { invites } = useUser({
    refetchInterval: 10 * 1000,
  });

  if (
    location.pathname === ROUTES.onboarding.invites ||
    location.pathname === ROUTES.onboarding.newTenant
  ) {
    if (
      !invites.loading &&
      invites.list.length > 0 &&
      !location.pathname.startsWith(ROUTES.onboarding.invites)
    ) {
      return <Navigate to={ROUTES.onboarding.invites} />;
    }

    if (
      !invites.loading &&
      invites.list.length === 0 &&
      !location.pathname.startsWith(ROUTES.onboarding.newTenant)
    ) {
      return <Navigate to={ROUTES.onboarding.newTenant} />;
    }
  }

  return (
    <div className="flex flex-col min-h-screen w-full">
      <header className="sticky top-0 z-50 w-full bg-background">
        <div className="flex h-16 items-center gap-2 border-b">
          <div className="flex w-full items-center justify-between px-4">
            <div className="flex items-center gap-2">
              <Logo variant="md" />
            </div>

            <div className="flex items-center gap-2">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    className="flex items-center gap-2 p-1 px-2"
                  >
                    <UserBlock userData={userData} />
                    <ChevronsUpDown className="ml-auto size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent
                  className="min-w-56 rounded-lg"
                  align="end"
                  sideOffset={4}
                >
                  <DropdownMenuLabel className="p-0 font-normal">
                    <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                      <UserBlock userData={userData} />
                    </div>
                  </DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuGroup>
                    <DropdownMenuItem onClick={() => toggleTheme()}>
                      {theme === 'dark' ? (
                        <Moon className="mr-2 h-4 w-4" />
                      ) : (
                        <Sun className="mr-2 h-4 w-4" />
                      )}
                      Toggle Theme
                    </DropdownMenuItem>
                  </DropdownMenuGroup>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={() => logout.mutate()}>
                    <LogOut className="mr-2 h-4 w-4" />
                    Log out
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
        </div>
      </header>
      <main className="flex flex-1 flex-col py-12 w-full items-center">
        <Outlet />
      </main>
    </div>
  );
}
