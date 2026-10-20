import { ThemeProvider, useTheme } from '@/components/hooks/use-theme';
import '@/index.css';
import type { Decorator, Preview } from '@storybook/react-vite';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useEffect } from 'react';

const THEME_STORAGE_KEY = 'storybook-ui-theme';

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

function ThemeSync({ theme }: { theme: 'dark' | 'light' }) {
  const { setTheme } = useTheme();

  useEffect(() => {
    setTheme(theme);
  }, [theme, setTheme]);

  return null;
}

const withAppProviders: Decorator = (Story, context) => {
  const theme = context.globals.theme === 'dark' ? 'dark' : 'light';
  localStorage.setItem(THEME_STORAGE_KEY, theme);

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider storageKey={THEME_STORAGE_KEY}>
        <ThemeSync theme={theme} />
        <div className="bg-background p-4 text-foreground">
          <Story />
        </div>
      </ThemeProvider>
    </QueryClientProvider>
  );
};

const preview: Preview = {
  globalTypes: {
    theme: {
      description: 'Color theme',
      toolbar: {
        title: 'Theme',
        icon: 'mirror',
        items: [
          { value: 'light', title: 'Light', icon: 'sun' },
          { value: 'dark', title: 'Dark', icon: 'moon' },
        ],
        dynamicTitle: true,
      },
    },
  },
  initialGlobals: {
    theme: 'dark',
  },
  decorators: [withAppProviders],
  parameters: {
    layout: 'fullscreen',
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
    a11y: {
      test: 'todo',
    },
  },
};

export default preview;
