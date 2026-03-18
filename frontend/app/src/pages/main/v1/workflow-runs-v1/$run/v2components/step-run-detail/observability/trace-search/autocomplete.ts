import {
  SPAN_STATUSES,
  SPAN_STATUS_COLORS,
  type TraceAutocompleteSuggestion,
  type TraceAutocompleteContext,
} from './types';

const STATUS_DESCRIPTIONS: Record<string, string> = {
  ok: 'Successful spans',
  error: 'Failed spans',
  unset: 'Spans without explicit status',
};

const STATIC_FILTER_KEYS: TraceAutocompleteSuggestion[] = [
  {
    type: 'key',
    label: 'status',
    value: 'status:',
    description: 'Filter by span status',
  },
];

export type AutocompleteMode = 'key' | 'value' | 'none';

export interface AutocompleteState {
  mode: AutocompleteMode;
  suggestions: TraceAutocompleteSuggestion[];
}

function buildFilterKeys(
  ctx: TraceAutocompleteContext,
): TraceAutocompleteSuggestion[] {
  const attrKeys: TraceAutocompleteSuggestion[] = ctx.attributeKeys.map(
    (key) => ({
      type: 'key' as const,
      label: key,
      value: `${key}:`,
      description: 'Span attribute',
    }),
  );
  return [...STATIC_FILTER_KEYS, ...attrKeys];
}

export function getTraceAutocomplete(
  query: string,
  ctx: TraceAutocompleteContext,
): AutocompleteState {
  const allKeys = buildFilterKeys(ctx);
  const trimmed = query.trimEnd();
  const lastWord = trimmed.split(' ').pop() || '';

  if (query.endsWith(' ') && !trimmed.endsWith(':')) {
    return { mode: 'key', suggestions: allKeys };
  }

  if (trimmed === '') {
    return { mode: 'key', suggestions: allKeys };
  }

  if (lastWord.startsWith('status:')) {
    const partial = lastWord.slice(7).toLowerCase();
    const suggestions = SPAN_STATUSES.filter((s) => s.startsWith(partial)).map(
      (s) => ({
        type: 'value' as const,
        label: s,
        value: s,
        description: STATUS_DESCRIPTIONS[s],
        color: SPAN_STATUS_COLORS[s],
      }),
    );
    return { mode: 'value', suggestions };
  }

  const colonIdx = lastWord.indexOf(':');
  if (colonIdx > 0) {
    const key = lastWord.slice(0, colonIdx);
    const partial = lastWord.slice(colonIdx + 1).toLowerCase();
    const knownValues = ctx.attributeValues.get(key) ?? [];
    const suggestions = knownValues
      .filter((v) => v.toLowerCase().startsWith(partial))
      .slice(0, 20)
      .map((v) => ({
        type: 'value' as const,
        label: v,
        value: v,
        description: key,
      }));
    return { mode: 'value', suggestions };
  }

  const matchingKeys = allKeys.filter(
    (k) => k.value.startsWith(lastWord.toLowerCase()) && lastWord.length > 0,
  );
  if (matchingKeys.length > 0) {
    return { mode: 'key', suggestions: matchingKeys };
  }

  return { mode: 'none', suggestions: [] };
}

export function applyTraceSuggestion(
  query: string,
  suggestion: TraceAutocompleteSuggestion,
): string {
  const trimmed = query.trimEnd();
  const words = trimmed.split(' ');
  const lastWord = words.pop() || '';

  if (suggestion.type === 'value') {
    const prefix = lastWord.slice(0, lastWord.indexOf(':') + 1);
    words.push(prefix + suggestion.value);
  } else {
    const allKeys = [...STATIC_FILTER_KEYS];
    const isPartialKey = allKeys.some((key) =>
      key.value.startsWith(lastWord.toLowerCase()),
    );
    if (lastWord && (isPartialKey || lastWord.length > 0)) {
      words.push(suggestion.value);
    } else {
      words.push(lastWord, suggestion.value);
    }
  }

  return words.filter(Boolean).join(' ');
}
