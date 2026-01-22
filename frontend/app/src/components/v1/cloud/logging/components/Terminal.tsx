import { LazyLog, ScrollFollow } from '@melloware/react-logviewer';
import { useTheme } from '@/components/hooks/use-theme';
import { useMemo, useCallback, useRef, useState, useEffect } from 'react';

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
  const containerRef = useRef<HTMLDivElement>(null);
  const [dimensions, setDimensions] = useState({ width: 0, height: 0 });

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const updateDimensions = () => {
      setDimensions({
        width: container.offsetWidth,
        height: container.offsetHeight,
      });
    };

    updateDimensions();

    const resizeObserver = new ResizeObserver(updateDimensions);
    resizeObserver.observe(container);

    return () => resizeObserver.disconnect();
  }, []);

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
      if (scrollableHeight <= 0) return;

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
      ref={containerRef}
      className={
        className ||
        'h-[500px] md:h-[600px] rounded-md relative overflow-hidden font-mono text-xs'
      }
      style={containerStyle}
    >
      {dimensions.height > 0 && dimensions.width > 0 && (
        <ScrollFollow
          startFollowing={autoScroll}
          render={({ follow, onScroll }) => (
            <LazyLog
              text={logs || ' '}
              follow={follow}
              height={dimensions.height}
              width={dimensions.width}
              onScroll={(args) => {
                onScroll(args);
                handleScroll(args);
              }}
              enableSearch={false}
              enableHotKeys={false}
              selectableLines={false}
              style={{
                background: 'transparent',
                fontFamily: 'Monaco, Menlo, "Courier New", monospace',
                fontSize: '11px',
              }}
              caseInsensitive
              extraLines={1}
            />
          )}
        />
      )}
    </div>
  );
}

export default Terminal;
