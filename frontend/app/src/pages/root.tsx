import { SidebarProvider } from '@/components/sidebar-provider';
import { ThemeProvider } from '@/components/theme-provider';
import { Toaster } from '@/components/v1/ui/toaster';
import { RefetchIntervalProvider } from '@/contexts/refetch-interval-context';
import { PropsWithChildren } from 'react';
import { Outlet } from '@tanstack/react-router';

function Root({ children }: PropsWithChildren) {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <RefetchIntervalProvider>
        <SidebarProvider>
          <div className="fixed h-full w-full">
            <Toaster />
            {children ?? <Outlet />}
          </div>
        </SidebarProvider>
      </RefetchIntervalProvider>
    </ThemeProvider>
  );
}

export default Root;
