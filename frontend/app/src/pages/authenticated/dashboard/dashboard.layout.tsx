import { Outlet } from 'react-router-dom';
import { SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar';
import { AppSidebar } from './components/sidebar/sidebar';
import { Separator } from '@/components/ui/separator';
import { BreadcrumbNav } from './components/breadcrumbs';
import useTenant from '@/hooks/use-tenant';
import { Unauthorized } from '@/components/errors/unauthorized';
import { BreadcrumbProvider } from '@/hooks/use-breadcrumbs';
import { ChevronsUpDown } from 'lucide-react';
import { UserBlock } from './components/sidebar/user-dropdown';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useTheme } from '@/components/theme-provider';
import { Sun, Moon, LogOut, BadgeCheck } from 'lucide-react';
import useUser from '@/hooks/use-user';
import { Button } from '@/components/ui/button';
import { useIsMobile } from '@/hooks/use-mobile';
import { Logo } from '@/components/ui/logo';

export default function Authenticated() {
  const { tenant, isLoading } = useTenant();
  const { toggleTheme, theme } = useTheme();
  const { logout } = useUser();
  const isMobile = useIsMobile();

  return (
    <>
      <BreadcrumbProvider>
        <SidebarProvider>
          <AppSidebar>
            <div className="flex flex-col h-screen">
              <div className="sticky top-0 z-50 w-full bg-background mt-3">
                <header className="flex h-16 items-center gap-2 border-b">
                  <div className="flex w-full items-center justify-between px-4">
                    <div className="flex items-center gap-2">
                      {isMobile && (
                        <SidebarTrigger
                          className="-ml-1"
                          icon={<Logo variant="md" />}
                        />
                      )}
                      <BreadcrumbNav />
                    </div>

                    <div className="flex items-center gap-2">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            className="flex items-center gap-2 p-1 px-2"
                          >
                            <UserBlock />
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
                              <UserBlock />
                            </div>
                          </DropdownMenuLabel>
                          <DropdownMenuSeparator />
                          <DropdownMenuGroup>
                            <DropdownMenuItem>
                              {/* TODO: Add account settings page */}
                              <BadgeCheck className="mr-2 h-4 w-4" />
                              Account Settings
                            </DropdownMenuItem>
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
                </header>
              </div>
              <main className="flex flex-1 flex-col gap-4 p-4 overflow-auto">
                {!isLoading && !tenant && <Unauthorized />}
                {!isLoading && tenant && <Outlet />}
              </main>
            </div>
          </AppSidebar>
        </SidebarProvider>
      </BreadcrumbProvider>
    </>
  );
}
