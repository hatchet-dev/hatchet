import { useTheme } from '@/components/hooks/use-theme';
import { cn } from '@/lib/utils';
import { LazyLog, ScrollFollow } from '@melloware/react-logviewer';
import { useMemo, useCallback, useRef } from 'react';

interface TerminalProps {
  logs: string;
  autoScroll?: boolean;
  onScrollToTop?: () => void;
  onScrollToBottom?: () => void;
  className?: string;
}

function Terminal({
  logs,
  autoScroll = false,
  onScrollToTop,
  onScrollToBottom,
  className,
}: TerminalProps) {
  const { theme: themeMode } = useTheme();
  const isDark = themeMode === 'dark';
  const lastScrollTopRef = useRef(0);

  const containerStyle = useMemo(
    () => ({
      background: isDark ? '#1e293b' : '#f8fafc',
      color: isDark ? '#dddddd' : '#1e293b',
    }),
    [isDark],
  );

  const handleScroll = useCallback(
    ({
      scrollTop,
      scrollHeight,
      clientHeight,
    }: {
      scrollTop: number;
      scrollHeight: number;
      clientHeight: number;
    }) => {
      const scrollableHeight = scrollHeight - clientHeight;
      if (scrollableHeight <= 0) {
        return;
      }

      const scrollPercentage = scrollTop / scrollableHeight;
      const isScrollingUp = scrollTop < lastScrollTopRef.current;
      const isScrollingDown = scrollTop > lastScrollTopRef.current;

      // Near top (newest logs with newest-first) - for running tasks
      if (isScrollingUp && scrollPercentage < 0.3 && onScrollToTop) {
        onScrollToTop();
      }

      // Near bottom (older logs with newest-first) - for infinite scroll
      if (isScrollingDown && scrollPercentage > 0.7 && onScrollToBottom) {
        onScrollToBottom();
      }

      lastScrollTopRef.current = scrollTop;
    },
    [onScrollToTop, onScrollToBottom],
  );

  return (
    <div
      className={cn(
        'terminal-root h-[500px] md:h-[600px] rounded-md w-full overflow-hidden',
        className,
      )}
      style={containerStyle}
    >
      <ScrollFollow
        startFollowing={autoScroll}
        render={({ follow, onScroll }) => (
          <LazyLog
            text={logs}
            follow={follow}
            onScroll={(args) => {
              onScroll(args);
              handleScroll(args);
            }}
          />
        )}
      />
    </div>
  );
}

export default Terminal;
