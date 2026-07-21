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

const SELECTABLE_THEMES: SelectableTheme[] = ['light', 'dark', 'system'];

const initialState: ThemeProviderState = {
  theme: 'system',
  setTheme: () => null,
  toggleTheme: () => null,
  currentlyVisibleTheme: 'light',
};

const ThemeProviderContext = createContext<ThemeProviderState>(initialState);

export const isSelectableTheme = (value: unknown): value is SelectableTheme =>
  typeof value === 'string' &&
  SELECTABLE_THEMES.includes(value as SelectableTheme);

const getThemeToDisplay = (theme: SelectableTheme): DisplayableTheme => {
  if (theme === 'system') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'light';
  }
  return theme;
};

const readStoredTheme = (
  storageKey: string,
  defaultTheme: SelectableTheme,
): SelectableTheme => {
  try {
    const stored = localStorage.getItem(storageKey);
    if (isSelectableTheme(stored)) {
      return stored;
    }
  } catch {
    // localStorage can throw in private mode / restricted iframes
  }
  return defaultTheme;
};

export function ThemeProvider({
  children,
  defaultTheme = 'system',
  storageKey = 'vite-ui-theme',
  ...props
}: ThemeProviderProps) {
  const [theme, setTheme] = useState<SelectableTheme>(() =>
    readStoredTheme(storageKey, defaultTheme),
  );

  const [currentlyVisibleTheme, setCurrentlyVisibleTheme] =
    useState<DisplayableTheme>(() => getThemeToDisplay(theme));

  useEffect(() => {
    const root = window.document.documentElement;

    root.classList.remove('light', 'dark');

    const themeToDisplay = getThemeToDisplay(theme);
    setCurrentlyVisibleTheme(themeToDisplay);
    root.classList.add(themeToDisplay);
    root.style.colorScheme = themeToDisplay;
  }, [theme]);

  // Follow OS theme changes while preference is "system"
  useEffect(() => {
    if (theme !== 'system') {
      return;
    }

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = () => {
      const themeToDisplay = getThemeToDisplay('system');
      setCurrentlyVisibleTheme(themeToDisplay);

      const root = window.document.documentElement;
      root.classList.remove('light', 'dark');
      root.classList.add(themeToDisplay);
      root.style.colorScheme = themeToDisplay;
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, [theme]);

  const setThemeAndLocal = (nextTheme: SelectableTheme) => {
    try {
      localStorage.setItem(storageKey, nextTheme);
    } catch {
      // ignore persistence failures; still update in-memory theme
    }
    setTheme(nextTheme);
  };

  const toggleTheme = () => {
    // Cycle based on what the user actually sees (handles "system" correctly).
    setThemeAndLocal(
      getThemeToDisplay(theme) === 'dark' ? 'light' : 'dark',
    );
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
