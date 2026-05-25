import { createContext, useContext, useEffect, useState } from 'react';

type DisplayableTheme = 'dark' | 'light';
type SelectableTheme = DisplayableTheme | 'system';

type ThemeProviderProps = {
  children: React.ReactNode;
  defaultTheme?: SelectableTheme;
  storageKey?: string;
};

type ThemeProviderState = {
  theme: SelectableTheme;
  setTheme: (theme: SelectableTheme) => void;
  toggleTheme: () => void;
  currentlyVisibleTheme: DisplayableTheme;
};

const initialState: ThemeProviderState = {
  theme: 'system',
  setTheme: () => null,
  toggleTheme: () => null,
  currentlyVisibleTheme: 'light',
};

const ThemeProviderContext = createContext<ThemeProviderState>(initialState);

const getThemeToDisplay = (theme: SelectableTheme): DisplayableTheme => {
  if (theme === 'system') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'light';
  }
  return theme;
};

export function ThemeProvider({
  children,
  defaultTheme = 'system',
  storageKey = 'vite-ui-theme',
  ...props
}: ThemeProviderProps) {
  const [theme, setTheme] = useState<SelectableTheme>(
    () => (localStorage.getItem(storageKey) as SelectableTheme) || defaultTheme,
  );

  const [currentlyVisibleTheme, setCurrentlyVisibleTheme] =
    useState<DisplayableTheme>(() => getThemeToDisplay(theme));

  useEffect(() => {
    const root = window.document.documentElement;

    root.classList.remove('light', 'dark');

    const themeToDisplay = getThemeToDisplay(theme);
    setCurrentlyVisibleTheme(themeToDisplay);
    root.classList.add(themeToDisplay);
  }, [theme]);

  const setThemeAndLocal = (theme: SelectableTheme) => {
    localStorage.setItem(storageKey, theme);
    setTheme(theme);
  };

  const toggleTheme = () => {
    setThemeAndLocal(theme === 'dark' ? 'light' : 'dark');
  };

  const value = {
    theme,
    setTheme: setThemeAndLocal,
    toggleTheme,
    currentlyVisibleTheme,
  };

  return (
    <ThemeProviderContext.Provider {...props} value={value}>
      {children}
    </ThemeProviderContext.Provider>
  );
}

export const useTheme = () => {
  const context = useContext(ThemeProviderContext);

  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }

  return context;
};
