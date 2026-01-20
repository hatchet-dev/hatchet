import {
  AutocompleteContext,
  AutocompleteSuggestion,
  RESERVED_FILTER_KEYS,
} from './types';

/**
 * Determine the autocomplete context based on query and cursor position
 */
export function getAutocompleteContext(
  query: string,
  cursorPosition: number,
): AutocompleteContext {
  // Get the text before cursor
  const beforeCursor = query.slice(0, cursorPosition);

  // Check if we're in the middle of typing a key:value pair
  const lastColonIndex = beforeCursor.lastIndexOf(':');
  const lastSpaceIndex = beforeCursor.lastIndexOf(' ');

  if (lastColonIndex > lastSpaceIndex) {
    // We're after a colon - suggest values
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

  // Check if we're typing a key (word without space after last space)
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

/**
 * Generate autocomplete suggestions based on context
 */
export function getSuggestions(
  context: AutocompleteContext,
  metadataKeys: string[],
  knownValues: Record<string, string[]> = {},
): AutocompleteSuggestion[] {
  const suggestions: AutocompleteSuggestion[] = [];

  if (context.mode === 'key') {
    const partial = (context.partialValue || '').toLowerCase();

    // Reserved keys first
    const reservedSuggestions: AutocompleteSuggestion[] = [
      {
        type: 'key',
        label: 'level',
        value: 'level:',
        description: 'Filter by log level (error, warn, info, debug)',
      },
      {
        type: 'key',
        label: 'after',
        value: 'after:',
        description: 'Logs after date/time (e.g., 2024-01-01, 1h, 2d)',
      },
      {
        type: 'key',
        label: 'before',
        value: 'before:',
        description: 'Logs before date/time',
      },
    ];

    // Add reserved keys that match
    for (const s of reservedSuggestions) {
      if (s.label.startsWith(partial)) {
        suggestions.push(s);
      }
    }

    // Add metadata keys that match
    for (const key of metadataKeys) {
      if (
        key.toLowerCase().startsWith(partial) &&
        !RESERVED_FILTER_KEYS.includes(key as (typeof RESERVED_FILTER_KEYS)[number])
      ) {
        suggestions.push({
          type: 'key',
          label: key,
          value: `${key}:`,
          description: 'Metadata filter',
        });
      }
    }
  } else if (context.mode === 'value' && context.currentKey) {
    const partial = (context.partialValue || '').toLowerCase();
    const key = context.currentKey;

    // Suggest known values for specific keys
    if (key === 'level') {
      const levels = ['error', 'warn', 'info', 'debug', 'trace'];
      for (const level of levels) {
        if (level.startsWith(partial)) {
          suggestions.push({
            type: 'value',
            label: level,
            value: level,
          });
        }
      }
    } else if (key === 'after' || key === 'before') {
      // Suggest relative time shortcuts
      const timeShortcuts = [
        { label: '1h', description: '1 hour ago' },
        { label: '6h', description: '6 hours ago' },
        { label: '1d', description: '1 day ago' },
        { label: '7d', description: '7 days ago' },
        { label: '30d', description: '30 days ago' },
      ];
      for (const shortcut of timeShortcuts) {
        if (shortcut.label.startsWith(partial)) {
          suggestions.push({
            type: 'value',
            label: shortcut.label,
            value: shortcut.label,
            description: shortcut.description,
          });
        }
      }
    } else if (knownValues[key]) {
      for (const value of knownValues[key]) {
        if (value.toLowerCase().startsWith(partial)) {
          suggestions.push({
            type: 'value',
            label: value,
            value: value.includes(' ') ? `"${value}"` : value,
          });
        }
      }
    }
  }

  return suggestions.slice(0, 10); // Limit to 10 suggestions
}
