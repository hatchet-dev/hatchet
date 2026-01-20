export const LOG_LEVELS = ['error', 'warn', 'info', 'debug'] as const;
export type LogLevel = (typeof LOG_LEVELS)[number];

export interface ParsedLogQuery {
  search?: string;
  level?: LogLevel;
  raw: string;
  isValid: boolean;
  errors: string[];
}

export interface AutocompleteSuggestion {
  type: 'key' | 'value';
  label: string;
  value: string;
  description?: string;
}

export interface LogSearchInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
}
