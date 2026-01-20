import { ReactNode } from 'react';

/**
 * Represents a single filter token in the parsed query
 */
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

/**
 * Reserved filter keys with special handling
 */
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

/**
 * Suggestion categories for grouped display
 */
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

/**
 * The parsed query output - this is what the component emits
 */
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

/**
 * Autocomplete suggestion types
 */
export type AutocompleteSuggestionType = 'key' | 'value';

export interface AutocompleteSuggestion {
  type: AutocompleteSuggestionType;
  label: string;
  value: string;
  description?: string;
  icon?: ReactNode;
  category?: SuggestionCategory;
}

/**
 * Autocomplete context - determines what suggestions to show
 */
export interface AutocompleteContext {
  mode: 'idle' | 'key' | 'value';
  currentKey?: string;
  partialValue?: string;
  cursorPosition: number;
}

/**
 * Props for the LogSearchInput component
 */
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
