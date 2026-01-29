import { cn } from '@/lib/utils';
import { LazyLog, ScrollFollow } from '@melloware/react-logviewer';
import { useCallback, useRef } from 'react';

interface TerminalProps {
  logs: string;
  autoScroll?: boolean;
  onScrollToTop?: () => void;
  onScrollToBottom?: () => void;
  onAtTopChange?: (atTop: boolean) => void;
  className?: string;
}

function Terminal({
  logs,
  autoScroll = false,
  onScrollToTop,
  onScrollToBottom,
  onAtTopChange,
  className,
}: TerminalProps) {
  const lastScrollTopRef = useRef(0);
  const wasAtTopRef = useRef(true);
  const wasInTopRegionRef = useRef(false);
  const wasInBottomRegionRef = useRef(false);

  const handleLineClick = useCallback(
    (event: React.MouseEvent<HTMLSpanElement>) => {
      const lineElement = (event.target as HTMLElement).closest('.log-line');
      if (lineElement) {
        lineElement.classList.toggle('expanded');
      }
    },
    [],
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

      const isAtTop = scrollPercentage < 0.05;
      if (onAtTopChange && isAtTop !== wasAtTopRef.current) {
        wasAtTopRef.current = isAtTop;
        onAtTopChange(isAtTop);
      }

      // Near top (newest logs with newest-first) - for running tasks
      // Only fire when entering the region (edge detection)
      // The region is defined by both scroll direction AND position, so changing
      // direction automatically resets the region state
      const isInTopRegion = isScrollingUp && scrollPercentage < 0.3;
      if (isInTopRegion && !wasInTopRegionRef.current && onScrollToTop) {
        onScrollToTop();
      }
      wasInTopRegionRef.current = isInTopRegion;

      // Near bottom (older logs with newest-first) - for infinite scroll
      // Only fire when entering the region (edge detection)
      // The region is defined by both scroll direction AND position, so changing
      // direction automatically resets the region state
      const isInBottomRegion = isScrollingDown && scrollPercentage > 0.7;
      if (
        isInBottomRegion &&
        !wasInBottomRegionRef.current &&
        onScrollToBottom
      ) {
        onScrollToBottom();
      }
      wasInBottomRegionRef.current = isInBottomRegion;

      lastScrollTopRef.current = scrollTop;
    },
    [onScrollToTop, onScrollToBottom, onAtTopChange],
  );

  return (
    <div
      className={cn(
        'terminal-root h-[500px] md:h-[600px] rounded-md w-full overflow-hidden',
        className,
      )}
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
            onLineContentClick={handleLineClick}
            selectableLines
          />
        )}
      />
    </div>
  );
}

export default Terminal;
