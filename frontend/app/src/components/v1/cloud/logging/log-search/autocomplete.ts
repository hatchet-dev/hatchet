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

const FILTER_KEYS: FilterSuggestion[] = [
  {
    type: 'key',
    label: 'level',
    value: 'level:',
    description: 'Filter by log level',
  },
  {
    type: 'key',
    label: 'attempt',
    value: 'attempt:',
    description: 'Filter by attempt number',
  },
];

export function getAutocomplete(
  query: string,
  availableAttempts: number[],
): AutocompleteState {
  const trimmed = query.trimEnd();
  const lastWord = trimmed.split(' ').pop() || '';

  if (query.endsWith(' ') && !trimmed.endsWith(':')) {
    return { mode: 'key', suggestions: FILTER_KEYS };
  }

  if (trimmed === '') {
    return { mode: 'key', suggestions: FILTER_KEYS };
  }

  if (lastWord.startsWith('level:')) {
    const partial = lastWord.slice(6).toLowerCase();
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

  if (lastWord.startsWith('attempt:')) {
    const partial = lastWord.slice(8);
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

  const matchingKeys = FILTER_KEYS.filter(
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
  return applyFilterSuggestion(query, suggestion, FILTER_KEYS);
}
