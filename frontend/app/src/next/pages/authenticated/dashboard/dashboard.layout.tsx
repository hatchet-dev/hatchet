import { Outlet } from 'react-router-dom';
import { SidebarProvider, SidebarTrigger } from '@/next/components/ui/sidebar';
import { AppSidebar } from './components/sidebar/sidebar';
import { Separator } from '@/next/components/ui/separator';
import { BreadcrumbNav } from './components/breadcrumbs';
import useTenant from '@/next/hooks/use-tenant';
import { Unauthorized } from '@/next/components/errors/unauthorized';
import { BreadcrumbProvider } from '@/next/hooks/use-breadcrumbs';
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
} from '@/next/components/ui/dropdown-menu';
import { useTheme } from '@/next/components/theme-provider';
import { Sun, Moon, LogOut } from 'lucide-react';
import useUser from '@/next/hooks/use-user';
import { Button } from '@/next/components/ui/button';
import { useIsMobile } from '@/next/hooks/use-mobile';
import { Logo } from '@/next/components/ui/logo';
import { Alerter } from './components/sidebar/alerter';
import { GrRevert } from 'react-icons/gr';

export default function DashboardLayout() {
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
                <header className="flex h-16 items-center gap-2 border-b px-4 md:px-8 lg:px-12">
                  <div className="flex w-full items-center justify-between">
                    <div className="flex items-center gap-2">
                      {isMobile && (
                        <>
                          <SidebarTrigger
                            className="-ml-1"
                            icon={<Logo variant="icon" />}
                          />
                          <Separator
                            orientation="vertical"
                            className="mr-2 h-4"
                          />
                        </>
                      )}
                      <BreadcrumbNav />
                    </div>

                    <div className="flex items-center gap-4">
                      {/* SECONDARY BUTTONS */}
                      <Alerter />
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            className="flex items-center gap-2 p-0"
                          >
                            <UserBlock variant="compact" />
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
                            <DropdownMenuItem
                              onClick={() => {
                                localStorage.setItem('next-ui', 'false');
                                window.location.href = '/';
                              }}
                            >
                              <GrRevert className="mr-2 h-4 w-4" />
                              Switch to Old UI
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
              <main className="flex flex-1 flex-col gap-4 overflow-auto">
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
