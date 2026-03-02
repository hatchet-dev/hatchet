import { AutocompleteSuggestion, LOG_LEVELS, LOG_LEVEL_COLORS } from './types';

const LEVEL_DESCRIPTIONS: Record<string, string> = {
  error: 'Error messages',
  warn: 'Warning messages',
  info: 'Informational messages',
  debug: 'Debug messages',
};

const FILTER_KEYS: AutocompleteSuggestion[] = [
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

export type AutocompleteMode = 'key' | 'value' | 'none';

export interface AutocompleteState {
  mode: AutocompleteMode;
  suggestions: AutocompleteSuggestion[];
}

export function getAutocomplete(
  query: string,
  availableAttempts: number[],
): AutocompleteState {
  const trimmed = query.trimEnd();
  const lastWord = trimmed.split(' ').pop() || '';

  // Check for trailing space FIRST - indicates user wants to add a new filter
  // Don't check this if the query ends with a colon (e.g., "level:")
  if (query.endsWith(' ') && !trimmed.endsWith(':')) {
    return { mode: 'key', suggestions: FILTER_KEYS };
  }

  // Check for empty input
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
  suggestion: AutocompleteSuggestion,
): string {
  const trimmed = query.trimEnd();
  const words = trimmed.split(' ');
  const lastWord = words.pop() || '';

  if (suggestion.type === 'value') {
    const prefix = lastWord.slice(0, lastWord.indexOf(':') + 1);
    words.push(prefix + suggestion.value);
  } else {
    const isPartialKey = FILTER_KEYS.some((key) =>
      key.value.startsWith(lastWord.toLowerCase()),
    );
    if (lastWord && isPartialKey) {
      words.push(suggestion.value);
    } else {
      words.push(lastWord, suggestion.value);
    }
  }

  return words.filter(Boolean).join(' ');
}
