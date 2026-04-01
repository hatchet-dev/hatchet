import { LOG_LEVELS, LOG_LEVEL_COLORS } from './types';
import {
  type FilterSuggestion,
  type AutocompleteState,
  applySuggestion as applyFilterSuggestion,
} from '@/components/v1/molecules/search-bar-with-filters/filter-query-utils';

export type { AutocompleteMode } from '@/components/v1/molecules/search-bar-with-filters/filter-query-utils';

const LEVEL_DESCRIPTIONS: Record<string, string> = {
  error: 'Error messages',
  warn: 'Warning messages',
  info: 'Informational messages',
  debug: 'Debug messages',
};

export const LOG_FILTER_KEYS = {
  LEVEL: 'level',
  ATTEMPT: 'attempt',
} as const;

export const STATIC_FILTER_KEYS: FilterSuggestion[] = [
  {
    type: 'key',
    label: 'level',
    value: `${LOG_FILTER_KEYS.LEVEL}:`,
    description: 'Filter by log level',
  },
  {
    type: 'key',
    label: 'attempt',
    value: `${LOG_FILTER_KEYS.ATTEMPT}:`,
    description: 'Filter by attempt number',
  },
  {
    type: 'key',
    label: 'workflow',
    value: 'workflow:',
    description: 'Filter by workflow name',
  },
];

export interface LogAutocompleteContext {
  availableAttempts?: number[];
  workflowNames?: string[];
}

export function getAutocomplete(
  query: string,
  context: LogAutocompleteContext,
): AutocompleteState {
  const { availableAttempts = [] } = context;
  const trimmed = query.trimEnd();
  const lastWord = trimmed.split(' ').pop() || '';

  if (query.endsWith(' ') && !trimmed.endsWith(':')) {
    return { mode: 'key', suggestions: STATIC_FILTER_KEYS };
  }

  if (trimmed === '') {
    return { mode: 'key', suggestions: STATIC_FILTER_KEYS };
  }

  const levelPrefix = `${LOG_FILTER_KEYS.LEVEL}:`;
  const attemptPrefix = `${LOG_FILTER_KEYS.ATTEMPT}:`;

  if (lastWord.startsWith(levelPrefix)) {
    const partial = lastWord.slice(levelPrefix.length).toLowerCase();
    const suggestions = LOG_LEVELS.filter((level) =>
      level.startsWith(partial),
    ).map((level) => ({
      type: 'value' as const,
      label: level,
      value: level,
      description: LEVEL_DESCRIPTIONS[level],
      color: LOG_LEVEL_COLORS[level],
    }));
    return { mode: 'value', suggestions };
  }

  if (lastWord.startsWith(attemptPrefix)) {
    const partial = lastWord.slice(attemptPrefix.length);
    const attempts = availableAttempts ?? [1, 2, 3];
    const suggestions = attempts
      .filter((attempt) => String(attempt).startsWith(partial))
      .map((attempt) => ({
        type: 'value' as const,
        label: String(attempt),
        value: String(attempt),
        description: `Attempt ${attempt}`,
      }));
    return { mode: 'value', suggestions };
  }

  const matchingKeys = STATIC_FILTER_KEYS.filter(
    (key) =>
      key.value.startsWith(lastWord.toLowerCase()) && lastWord.length > 0,
  );
  if (matchingKeys.length > 0) {
    return { mode: 'key', suggestions: matchingKeys };
  }

  return { mode: 'none', suggestions: [] };
}

export function applySuggestion(
  query: string,
  suggestion: FilterSuggestion,
): string {
  return applyFilterSuggestion(query, suggestion, STATIC_FILTER_KEYS);
}
