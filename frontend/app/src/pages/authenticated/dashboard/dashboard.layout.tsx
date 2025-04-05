import { Outlet } from 'react-router-dom';
import { SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar';
import { AppSidebar } from './components/sidebar/sidebar';
import { Separator } from '@/components/ui/separator';
import { AppBreadcrumbs } from './components/breadcrumbs';
import useTenant from '@/hooks/use-tenant';
import { Unauthorized } from '@/components/errors/unauthorized';

export default function Authenticated() {
  const { tenant } = useTenant();

  return (
    <>
      <SidebarProvider>
        <AppSidebar>
          <header className="flex h-16 shrink-0 items-center gap-2">
            <div className="flex items-center gap-2 px-4">
              <SidebarTrigger className="-ml-1" />
              <Separator orientation="vertical" className="mr-2 h-4" />
              <AppBreadcrumbs />
            </div>
          </header>
          <main className="flex flex-1 flex-col gap-4 p-4 pt-0">
            {JSON.stringify(tenant)}
            {tenant ? <Outlet /> : <Unauthorized />}
          </main>
        </AppSidebar>
      </SidebarProvider>
    </>
  );
}
