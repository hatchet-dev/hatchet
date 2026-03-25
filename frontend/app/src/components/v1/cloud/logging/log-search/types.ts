import { V1LogLineLevel } from '@/lib/api';

export type { FilterSuggestion as AutocompleteSuggestion } from '@/components/v1/molecules/search-bar-with-filters/filter-query-utils';

export const LOG_LEVELS = ['error', 'warn', 'info', 'debug'] as const;
export type LogLevel = (typeof LOG_LEVELS)[number];

export const LOG_LEVEL_TO_API: Record<LogLevel, V1LogLineLevel> = {
  error: V1LogLineLevel.ERROR,
  warn: V1LogLineLevel.WARN,
  info: V1LogLineLevel.INFO,
  debug: V1LogLineLevel.DEBUG,
};

export const LOG_LEVEL_COLORS: Record<LogLevel, string> = {
  error: 'bg-red-500',
  warn: 'bg-yellow-500',
  info: 'bg-green-500',
  debug: 'bg-slate-500',
};

export interface ParsedLogQuery {
  search?: string;
  level?: LogLevel;
  attempt?: number;
  raw: string;
  isValid: boolean;
  errors: string[];
}

export interface LogSearchInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
}
