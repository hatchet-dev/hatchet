import {
  AutocompleteContext,
  AutocompleteSuggestion,
  LOG_LEVELS,
} from './types';

const LEVEL_DESCRIPTIONS: Record<string, string> = {
  error: 'Error messages',
  warn: 'Warning messages',
  info: 'Informational messages',
  debug: 'Debug messages',
};

export function getAutocompleteContext(
  query: string,
  cursorPosition: number,
): AutocompleteContext {
  const beforeCursor = query.slice(0, cursorPosition);

  const lastColonIndex = beforeCursor.lastIndexOf(':');
  const lastSpaceIndex = beforeCursor.lastIndexOf(' ');

  if (lastColonIndex > lastSpaceIndex) {
    const keyStart = lastSpaceIndex + 1;
    const key = beforeCursor.slice(keyStart, lastColonIndex);
    const partialValue = beforeCursor.slice(lastColonIndex + 1);

    return {
      mode: 'value',
      currentKey: key.toLowerCase(),
      partialValue,
      cursorPosition,
    };
  }

  const lastWord = beforeCursor.slice(lastSpaceIndex + 1);
  if (lastWord.length > 0 && !lastWord.includes(':')) {
    return {
      mode: 'key',
      partialValue: lastWord,
      cursorPosition,
    };
  }

  return {
    mode: 'idle',
    cursorPosition,
  };
}

export function getSuggestions(
  context: AutocompleteContext,
): AutocompleteSuggestion[] {
  if (context.mode === 'idle') {
    return [
      {
        type: 'key',
        label: 'level',
        value: 'level:',
        description: 'Filter by log level',
      },
    ];
  }

  if (context.mode === 'key') {
    const partial = (context.partialValue || '').toLowerCase();
    if ('level'.startsWith(partial)) {
      return [
        {
          type: 'key',
          label: 'level',
          value: 'level:',
          description: 'Filter by log level',
        },
      ];
    }
    return [];
  }

  if (context.mode === 'value' && context.currentKey === 'level') {
    const partial = (context.partialValue || '').toLowerCase();
    return LOG_LEVELS.filter((level) => level.startsWith(partial)).map(
      (level) => ({
        type: 'value' as const,
        label: level,
        value: level,
        description: LEVEL_DESCRIPTIONS[level],
      }),
    );
  }

  return [];
}
