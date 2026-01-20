export interface QueryToken {
  type: 'filter' | 'text';
  key?: string;
  value: string;
  raw: string;
  position: {
    start: number;
    end: number;
  };
}

export const LOG_LEVELS = ['error', 'warn', 'info', 'debug'] as const;
export type LogLevel = (typeof LOG_LEVELS)[number];

export interface ParsedLogQuery {
  search?: string;
  level?: LogLevel;
  tokens: QueryToken[];
  raw: string;
  isValid: boolean;
  errors: string[];
}

export type AutocompleteSuggestionType = 'key' | 'value';

export interface AutocompleteSuggestion {
  type: AutocompleteSuggestionType;
  label: string;
  value: string;
  description?: string;
}

export interface AutocompleteContext {
  mode: 'idle' | 'key' | 'value';
  currentKey?: string;
  partialValue?: string;
  cursorPosition: number;
}

export interface LogSearchInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  showAutocomplete?: boolean;
  className?: string;
}
