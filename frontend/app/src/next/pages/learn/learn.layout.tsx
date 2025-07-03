import { Outlet } from 'react-router-dom';
import { ChevronsUpDown, Sun, Moon, LogOut, UserIcon } from 'lucide-react';
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
import { SignInRequiredAction } from './components/signin-required-action';
import { User } from '@/lib/api';

export const UserBlock = ({ userData }: { userData: User | undefined }) => (
  <div className="flex items-center gap-2">
    <div className="flex size-6 items-center justify-center rounded-full bg-primary text-primary-foreground">
      {userData?.email?.[0]?.toUpperCase() || <UserIcon className="w-4 h-4" />}
    </div>
    <div className="flex flex-col">
      <span className="text-xs font-medium">{userData?.email || 'User'}</span>
    </div>
  </div>
);

export function LearnLayout() {
  const { toggleTheme, theme } = useTheme();
  const { logout, data: userData } = useUser();

  // Simple user block component similar to dashboard's UserBlock

  return (
    <div className="flex flex-col min-h-screen w-full">
      <header className="sticky top-0 z-50 w-full bg-background">
        <div className="flex h-16 items-center gap-2 border-b">
          <div className="flex w-full items-center justify-between px-4">
            <div className="flex items-center gap-2">
              <Logo variant="md" />
            </div>

            <div className="flex items-center gap-2">
              <SignInRequiredAction>
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
              </SignInRequiredAction>
            </div>
          </div>
        </div>
      </header>
      <main className="flex flex-1 flex-col p-0">
        <Outlet />
      </main>
    </div>
  );
}
