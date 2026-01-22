import Terminal from './components/Terminal';
import { LogLine } from './log-search/use-logs';
import { useMemo } from 'react';

const DATE_FORMAT_OPTIONS: Intl.DateTimeFormatOptions = {
  year: 'numeric',
  month: 'numeric',
  day: 'numeric',
  hour: 'numeric',
  minute: 'numeric',
  second: 'numeric',
};

// ANSI color codes
const WHITE = '\x1b[37m'; // Regular white for timestamps (dimmer than bright white)
const RESET = '\x1b[0m'; // Reset to default

// Nice theme colors for instance names (using bright ANSI colors)
const INSTANCE_COLORS = [
  '\x1b[94m', // Bright blue
  '\x1b[96m', // Bright cyan
  '\x1b[95m', // Bright magenta
  '\x1b[36m', // Cyan
  '\x1b[34m', // Blue
  '\x1b[35m', // Magenta
];

// Simple hash function for stable color assignment
const hashString = (str: string): number => {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = (hash << 5) - hash + char;
    hash = hash & hash; // Convert to 32bit integer
  }
  return Math.abs(hash);
};

const getInstanceColor = (instance: string): string => {
  const hash = hashString(instance);
  return INSTANCE_COLORS[hash % INSTANCE_COLORS.length];
};

const formatLogLine = (log: LogLine): string => {
  let line = '';

  if (log.timestamp) {
    const formattedTime = new Date(log.timestamp)
      .toLocaleString('sv', DATE_FORMAT_OPTIONS)
      .replace(',', '.')
      .replace(' ', 'T');
    line += `${WHITE}${formattedTime}${RESET} `;
  }

  if (log.instance) {
    const color = getInstanceColor(log.instance);
    line += `${color}[${log.instance}]${RESET} `;
  }

  line += log.line || '';

  return line;
};

export interface LogViewerProps {
  logs: LogLine[];
  onScroll?: (scrollData: {
    scrollTop: number;
    scrollHeight: number;
    clientHeight: number;
  }) => void;
  emptyMessage?: string;
}

export function LogViewer({
  logs,
  onScroll,
  emptyMessage = 'Waiting for logs...',
}: LogViewerProps) {
  const formattedLogs = useMemo(() => {
    const showLogs =
      logs.length > 0
        ? logs
        : [
            {
              line: emptyMessage,
              timestamp: new Date().toISOString(),
              instance: 'Hatchet',
            },
          ];

    const sortedLogs = [...showLogs].sort((a, b) => {
      if (!a.timestamp || !b.timestamp) {
        return 0;
      }
      // Oldest first, newest at bottom (ascending order)
      return new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime();
    });

    return sortedLogs.map(formatLogLine).join('\n');
  }, [logs, emptyMessage]);

  // Memoize callbacks object to prevent unnecessary recreations
  const callbacks = useMemo(() => {
    const noOp = () => {};
    return {
      onTopReached: noOp,
      onBottomReached: noOp,
      onInfiniteScroll: onScroll || noOp,
    };
  }, [onScroll]);

  return (
    <Terminal
      logs={formattedLogs}
      callbacks={callbacks}
      className="max-h-[25rem] min-h-[25rem] pl-6 pt-6 pb-6 rounded-md relative overflow-hidden font-mono text-xs [&_canvas]:block [&_.xterm-cursor]:!hidden [&_textarea]:!fixed [&_textarea]:!left-[-9999px] [&_textarea]:!top-[-9999px]"
    />
  );
}
