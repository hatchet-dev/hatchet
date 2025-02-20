import { SidebarProvider } from '@/components/sidebar-provider';
import { ThemeProvider } from '@/components/theme-provider';
import { Toaster } from '@/components/ui/toaster';
import { PropsWithChildren } from 'react';
import { Outlet } from 'react-router-dom';

function Root({ children }: PropsWithChildren) {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <SidebarProvider>
        <div className="fixed h-full w-full">
          <Toaster />
          {children ?? <Outlet />}
        </div>
      </SidebarProvider>
    </ThemeProvider>
  );
}

export default Root;
