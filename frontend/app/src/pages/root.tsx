import { ThemeProvider } from '@/components/theme-provider';
import { PropsWithChildren } from 'react';
import { Outlet } from 'react-router-dom';

function Root({ children }: PropsWithChildren) {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      {children ?? <Outlet />}
    </ThemeProvider>
  );
}

export default Root;
