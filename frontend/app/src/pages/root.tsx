import { ThemeProvider } from '@/components/theme-provider';
import { Toaster } from '@/components/ui/toaster';
import { Outlet } from 'react-router-dom';

function Root() {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <div className="fixed h-full w-full">
        <Toaster />
        <Outlet />
      </div>
    </ThemeProvider>
  );
}

export default Root;
