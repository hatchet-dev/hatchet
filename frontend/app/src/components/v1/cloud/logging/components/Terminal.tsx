import { useTerminal, TerminalScrollCallbacks } from '../hooks/useTerminal';
import { darkTheme, lightTheme } from '../terminalThemes';
import { useTheme } from '@/components/hooks/use-theme';
import { useRef, useState, useMemo } from 'react';

interface TerminalProps {
  logs: string;
  autoScroll?: boolean;
  callbacks?: TerminalScrollCallbacks;
  className?: string;
}

function Terminal({ logs, autoScroll, callbacks, className }: TerminalProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [initFailed, setInitFailed] = useState(false);
  const { theme: themeMode } = useTheme();

  // Determine if dark mode
  const isDark =
    themeMode === 'dark' ||
    (themeMode === 'system' &&
      window.matchMedia('(prefers-color-scheme: dark)').matches);

  // Memoize theme to prevent unnecessary reinitializations
  const theme = useMemo(() => (isDark ? darkTheme : lightTheme), [isDark]);

  // Memoize options to prevent unnecessary effect triggers
  const options = useMemo(
    () => ({
      autoScroll,
      callbacks,
      theme,
      onInitError: () => setInitFailed(true),
    }),
    [autoScroll, callbacks, theme],
  );

  useTerminal(containerRef, logs, options);

  // Fallback: render plain text if terminal init failed
  if (initFailed) {
    // Strip ANSI codes for plain text display
    // eslint-disable-next-line no-control-regex
    const plainText = logs.replace(/\x1b\[[0-9;]*m/g, '');

    return (
      <div
        className={
          className ||
          'h-[500px] md:h-[600px] pl-6 pt-6 pb-6 bg-muted rounded-md overflow-auto font-mono text-xs whitespace-pre-wrap'
        }
        style={{ color: theme.foreground }}
      >
        {plainText}
      </div>
    );
  }

  return (
    <div
      className={
        className ||
        'h-[500px] md:h-[600px] pl-6 pt-6 pb-6 rounded-md relative overflow-hidden font-mono text-xs [&_canvas]:block [&_.xterm-cursor]:!hidden [&_textarea]:!fixed [&_textarea]:!left-[-9999px] [&_textarea]:!top-[-9999px]'
      }
      style={{ backgroundColor: theme.background }}
      ref={containerRef}
      onFocus={(e) => e.currentTarget.blur()}
      tabIndex={-1}
    ></div>
  );
}

export default Terminal;
