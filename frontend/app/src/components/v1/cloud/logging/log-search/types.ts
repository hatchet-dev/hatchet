import { ReactNode } from 'react';

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

export const RESERVED_FILTER_KEYS = [
  'after',
  'before',
  'level',
  'worker',
  'workflow',
  'step',
  'run',
] as const;
export type ReservedFilterKey = (typeof RESERVED_FILTER_KEYS)[number];

export type SuggestionCategory =
  | 'time'
  | 'log-attributes'
  | 'workflow'
  | 'metadata';

export const SUGGESTION_CATEGORY_LABELS: Record<SuggestionCategory, string> = {
  time: 'Time Filters',
  'log-attributes': 'Log Attributes',
  workflow: 'Workflow',
  metadata: 'Metadata',
};

export interface ParsedLogQuery {
  search?: string;
  after?: string;
  before?: string;
  level?: string;
  metadata: Record<string, string>;
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
  icon?: ReactNode;
  category?: SuggestionCategory;
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
  onQueryChange: (query: ParsedLogQuery) => void;
  metadataKeys: string[];
  placeholder?: string;
  showAutocomplete?: boolean;
  className?: string;
  knownValues?: Record<string, string[]>;
}
