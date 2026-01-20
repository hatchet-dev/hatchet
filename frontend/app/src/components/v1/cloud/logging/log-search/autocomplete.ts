import {
  AutocompleteContext,
  AutocompleteSuggestion,
  RESERVED_FILTER_KEYS,
  SuggestionCategory,
} from './types';


interface FilterKeyDefinition {
  key: string;
  description: string;
  category: SuggestionCategory;
  values?: { label: string; description?: string }[];
}

const FILTER_KEY_DEFINITIONS: FilterKeyDefinition[] = [
  {
    key: 'level',
    description: 'Filter by log level',
    category: 'log-attributes',
    values: [
      { label: 'error', description: 'Error messages' },
      { label: 'warn', description: 'Warning messages' },
      { label: 'info', description: 'Informational messages' },
      { label: 'debug', description: 'Debug messages' },
      { label: 'trace', description: 'Trace messages' },
    ],
  },
  {
    key: 'after',
    description: 'Logs after date/time',
    category: 'time',
    values: [
      { label: '15m', description: '15 minutes ago' },
      { label: '1h', description: '1 hour ago' },
      { label: '6h', description: '6 hours ago' },
      { label: '1d', description: '1 day ago' },
      { label: '7d', description: '7 days ago' },
      { label: '30d', description: '30 days ago' },
    ],
  },
  {
    key: 'before',
    description: 'Logs before date/time',
    category: 'time',
    values: [
      { label: '15m', description: '15 minutes ago' },
      { label: '1h', description: '1 hour ago' },
      { label: '6h', description: '6 hours ago' },
      { label: '1d', description: '1 day ago' },
      { label: '7d', description: '7 days ago' },
    ],
  },
];

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

function getFilterDefinition(key: string): FilterKeyDefinition | undefined {
  return FILTER_KEY_DEFINITIONS.find((def) => def.key === key);
}

export function getSuggestions(
  context: AutocompleteContext,
  metadataKeys: string[],
  knownValues: Record<string, string[]> = {},
): AutocompleteSuggestion[] {
  const suggestions: AutocompleteSuggestion[] = [];

  if (context.mode === 'idle') {
    for (const def of FILTER_KEY_DEFINITIONS) {
      suggestions.push({
        type: 'key',
        label: def.key,
        value: `${def.key}:`,
        description: def.description,
        category: def.category,
      });
    }

    for (const key of metadataKeys) {
      if (
        !RESERVED_FILTER_KEYS.includes(
          key as (typeof RESERVED_FILTER_KEYS)[number],
        )
      ) {
        suggestions.push({
          type: 'key',
          label: key,
          value: `${key}:`,
          description: 'Metadata filter',
          category: 'metadata',
        });
      }
    }
  } else if (context.mode === 'key') {
    const partial = (context.partialValue || '').toLowerCase();

    for (const def of FILTER_KEY_DEFINITIONS) {
      if (def.key.startsWith(partial)) {
        suggestions.push({
          type: 'key',
          label: def.key,
          value: `${def.key}:`,
          description: def.description,
          category: def.category,
        });
      }
    }

    for (const key of metadataKeys) {
      if (
        key.toLowerCase().startsWith(partial) &&
        !RESERVED_FILTER_KEYS.includes(
          key as (typeof RESERVED_FILTER_KEYS)[number],
        )
      ) {
        suggestions.push({
          type: 'key',
          label: key,
          value: `${key}:`,
          description: 'Metadata filter',
          category: 'metadata',
        });
      }
    }
  } else if (context.mode === 'value' && context.currentKey) {
    const partial = (context.partialValue || '').toLowerCase();
    const key = context.currentKey;
    const filterDef = getFilterDefinition(key);

    if (filterDef?.values) {
      for (const val of filterDef.values) {
        if (val.label.toLowerCase().startsWith(partial)) {
          suggestions.push({
            type: 'value',
            label: val.label,
            value: val.label,
            description: val.description,
            category: filterDef.category,
          });
        }
      }
    }

    if (knownValues[key]) {
      const existingLabels = new Set(suggestions.map((s) => s.label));
      for (const value of knownValues[key]) {
        if (
          value.toLowerCase().startsWith(partial) &&
          !existingLabels.has(value)
        ) {
          suggestions.push({
            type: 'value',
            label: value,
            value: value.includes(' ') ? `"${value}"` : value,
            category: filterDef?.category || 'metadata',
          });
        }
      }
    }
  }

  return suggestions.slice(0, 15); // Limit to 15 suggestions
}

export function groupSuggestionsByCategory(
  suggestions: AutocompleteSuggestion[],
): Map<SuggestionCategory , AutocompleteSuggestion[]> {
  const groups = new Map<SuggestionCategory , AutocompleteSuggestion[]>();

  const categoryOrder: (SuggestionCategory)[] = [
    'log-attributes',
    'time',
    'metadata',
  ];

  for (const cat of categoryOrder) {
    groups.set(cat, []);
  }

  for (const suggestion of suggestions) {
    const category = suggestion.category;
    const group = category ? groups.get(category) : undefined;
    if (group) {
      group.push(suggestion);
    }
  }

  for (const [cat, items] of groups) {
    if (items.length === 0) {
      groups.delete(cat);
    }
  }

  return groups;
}
