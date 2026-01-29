import Terminal from './components/Terminal';
import { LogLine } from './log-search/use-logs';
import { V1TaskStatus } from '@/lib/api';
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
  onScrollToBottom?: () => void;
  onScrollToTop?: () => void;
  onAtTopChange?: (atTop: boolean) => void;
  isLoading?: boolean;
  taskStatus?: V1TaskStatus;
}

function getEmptyStateMessage(taskStatus?: V1TaskStatus): string {
  switch (taskStatus) {
    case V1TaskStatus.COMPLETED:
      return 'Task completed with no logs.';
    case V1TaskStatus.FAILED:
      return 'Task failed with no logs.';
    case V1TaskStatus.CANCELLED:
      return 'Task was cancelled with no logs.';
    case V1TaskStatus.RUNNING:
    case V1TaskStatus.QUEUED:
      return 'Waiting for logs...';
    default:
      return 'No logs available.';
  }
}

export function LogViewer({
  logs,
  onScrollToBottom,
  onScrollToTop,
  onAtTopChange,
  isLoading,
  taskStatus,
}: LogViewerProps) {
  const formattedLogs = useMemo(() => {
    if (logs.length === 0) {
      return '';
    }

    const sortedLogs = [...logs].sort((a, b) => {
      if (!a.timestamp || !b.timestamp) {
        return 0;
      }
      // descending
      return new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime();
    });

    return sortedLogs.map(formatLogLine).join('\n');
  }, [logs]);

  const isRunning = taskStatus === V1TaskStatus.RUNNING;

  if (isLoading) {
    return (
      <div className="max-h-[25rem] min-h-[25rem] rounded-md relative overflow-hidden bg-[var(--terminal-bg)] flex items-center justify-center">
        <span className="text-sm text-muted-foreground">Loading logs...</span>
      </div>
    );
  }

  const isEmpty = logs.length === 0;
  if (isEmpty && taskStatus !== undefined) {
    return (
      <div className="max-h-[25rem] min-h-[25rem] rounded-md relative overflow-hidden bg-[var(--terminal-bg)] flex items-center justify-center">
        <span className="text-sm text-muted-foreground">
          {getEmptyStateMessage(taskStatus)}
        </span>
      </div>
    );
  }

  return (
    <div className="relative">
      {isRunning && (
        <div className="absolute top-2 right-4 z-10 flex items-center gap-2 text-xs text-muted-foreground bg-[var(--terminal-bg)]/80 px-2 py-1 rounded">
          <span className="relative flex h-2 w-2">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
          </span>
          <span>Live</span>
        </div>
      )}
      <Terminal
        logs={formattedLogs}
        onScrollToTop={onScrollToTop}
        onScrollToBottom={onScrollToBottom}
        onAtTopChange={onAtTopChange}
        className="max-h-[25rem] min-h-[25rem] rounded-md relative overflow-hidden"
      />
    </div>
  );
}
