import { ThemeProvider } from '@/components/hooks/use-theme';
import '@/index.css';
import type { Decorator, Preview } from '@storybook/react-vite';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const THEME_STORAGE_KEY = 'storybook-ui-theme';

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

// Drive the app's own ThemeProvider from the toolbar so components using
// useTheme() (code highlighter, editor) follow the selected theme too.
const withAppProviders: Decorator = (Story, context) => {
  const theme = context.globals.theme === 'dark' ? 'dark' : 'light';
  localStorage.setItem(THEME_STORAGE_KEY, theme);

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider key={theme} storageKey={THEME_STORAGE_KEY}>
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
