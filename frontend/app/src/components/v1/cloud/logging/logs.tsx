import Terminal from './components/Terminal';
import React, { useMemo } from 'react';

export interface ExtendedLogLine {
  /** @format date-time */
  timestamp?: string;
  instance?: string;
  line: string;
}

type LogProps = {
  logs: ExtendedLogLine[];
  onScrollToTop?: () => void;
  onScrollToBottom?: () => void;
  autoScroll?: boolean;
};

const options: Intl.DateTimeFormatOptions = {
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

const formatLogLine = (log: ExtendedLogLine): string => {
  let line = '';

  // Add timestamp
  if (log.timestamp) {
    const formattedTime = new Date(log.timestamp)
      .toLocaleString('sv', options)
      .replace(',', '.')
      .replace(' ', 'T');
    line += `${WHITE}${formattedTime}${RESET} `;
  }

  // Add instance name with stable color
  if (log.instance) {
    const color = getInstanceColor(log.instance);
    line += `${color}[${log.instance}]${RESET} `;
  }

  // Add the actual log line (preserving any ANSI codes it already has)
  line += log.line || '';

  return line;
};

const LoggingComponent: React.FC<LogProps> = ({
  logs,
  onScrollToTop,
  onScrollToBottom,
  autoScroll = true,
}) => {
  const formattedLogs = useMemo(() => {
    const showLogs =
      logs.length > 0
        ? logs
        : [
            {
              line: 'Waiting for logs...',
              timestamp: new Date().toISOString(),
              instance: 'Hatchet',
            },
          ];

    // Sort by timestamp (newest first at top)
    const sortedLogs = [...showLogs].sort((a, b) => {
      if (!a.timestamp || !b.timestamp) {
        return 0;
      }
      return new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime();
    });

    // Convert to terminal format
    return sortedLogs.map(formatLogLine).join('\n');
  }, [logs]);

  return (
    <Terminal
      logs={formattedLogs}
      autoScroll={autoScroll}
      onScrollToTop={onScrollToTop}
      onScrollToBottom={onScrollToBottom}
    />
  );
};

export default LoggingComponent;
