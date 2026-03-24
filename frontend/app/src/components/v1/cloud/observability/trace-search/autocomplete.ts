import {
  SPAN_STATUSES,
  SPAN_STATUS_COLORS,
  type TraceAutocompleteContext,
} from './types';
import {
  type FilterSuggestion,
  type AutocompleteState,
  applySuggestion as applyFilterSuggestion,
} from '@/components/v1/molecules/search-bar-with-filters/filter-query-utils';

export type { AutocompleteMode } from '@/components/v1/molecules/search-bar-with-filters/filter-query-utils';

const STATUS_DESCRIPTIONS: Record<string, string> = {
  ok: 'Successful spans',
  error: 'Failed spans',
  unset: 'Spans without explicit status',
};

export const FILTER_KEYS = {
  STATUS: 'status',
  SPAN_NAME: 'span-name',
} as const;

export const STATIC_FILTER_KEYS: FilterSuggestion[] = [
  {
    type: 'key',
    label: 'status',
    value: `${FILTER_KEYS.STATUS}:`,
    description: 'Filter by span status',
  },
  {
    type: 'key',
    label: 'span name',
    value: `${FILTER_KEYS.SPAN_NAME}:`,
    description: 'Filter by span name',
  },
];

function buildFilterKeys(ctx: TraceAutocompleteContext): FilterSuggestion[] {
  const attrKeys: FilterSuggestion[] = ctx.attributeKeys.map((key) => ({
    type: 'key' as const,
    label: key,
    value: `${key}:`,
    description: 'Span attribute',
  }));
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

  const statusPrefix = `${FILTER_KEYS.STATUS}:`;
  const spanNamePrefix = `${FILTER_KEYS.SPAN_NAME}:`;

  if (lastWord.startsWith(statusPrefix)) {
    const partial = lastWord.slice(statusPrefix.length).toLowerCase();
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

  if (lastWord.startsWith(spanNamePrefix)) {
    const partial = lastWord.slice(spanNamePrefix.length).toLowerCase();
    const suggestions = ctx.spanNames
      .filter((n) => n.toLowerCase().includes(partial))
      .slice(0, 20)
      .map((n) => ({
        type: 'value' as const,
        label: n,
        value: n,
        description: 'Span name',
      }));
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
  suggestion: FilterSuggestion,
): string {
  return applyFilterSuggestion(query, suggestion, STATIC_FILTER_KEYS);
}
