import { AutocompleteSuggestion, LOG_LEVELS } from './types';

const LEVEL_DESCRIPTIONS: Record<string, string> = {
  error: 'Error messages',
  warn: 'Warning messages',
  info: 'Informational messages',
  debug: 'Debug messages',
};

export type AutocompleteMode = 'key' | 'value' | 'none';

export interface AutocompleteState {
  mode: AutocompleteMode;
  suggestions: AutocompleteSuggestion[];
}

export function getAutocomplete(query: string): AutocompleteState {
  const trimmed = query.trimEnd();
  const lastWord = trimmed.split(' ').pop() || '';

  if (lastWord.startsWith('level:')) {
    const partial = lastWord.slice(6).toLowerCase();
    const suggestions = LOG_LEVELS.filter((level) =>
      level.startsWith(partial),
    ).map((level) => ({
      type: 'value' as const,
      label: level,
      value: level,
      description: LEVEL_DESCRIPTIONS[level],
    }));
    return { mode: 'value', suggestions };
  }

  if ('level:'.startsWith(lastWord.toLowerCase()) && lastWord.length > 0) {
    return {
      mode: 'key',
      suggestions: [
        {
          type: 'key',
          label: 'level',
          value: 'level:',
          description: 'Filter by log level',
        },
      ],
    };
  }

  if (trimmed === '' || query.endsWith(' ')) {
    return {
      mode: 'key',
      suggestions: [
        {
          type: 'key',
          label: 'level',
          value: 'level:',
          description: 'Filter by log level',
        },
      ],
    };
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
    if (lastWord && 'level:'.startsWith(lastWord.toLowerCase())) {
      words.push(suggestion.value);
    } else {
      words.push(lastWord, suggestion.value);
    }
  }

  return words.filter(Boolean).join(' ');
}
