import { SidebarProvider } from '@/components/sidebar-provider';
import { ThemeProvider } from '@/components/theme-provider';
import { Toaster } from '@/components/v1/ui/toaster';
import { RefetchIntervalProvider } from '@/contexts/refetch-interval-context';
import { Outlet } from '@tanstack/react-router';
import { PropsWithChildren } from 'react';

function Root({ children }: PropsWithChildren) {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <RefetchIntervalProvider>
        <SidebarProvider>
          {/* Root should not own scrolling; route shells decide their scroll behavior. */}
          <div className="h-full w-full overflow-hidden">
            <Toaster />
            {children ?? <Outlet />}
          </div>
        </SidebarProvider>
      </RefetchIntervalProvider>
    </ThemeProvider>
  );
}

export default Root;
