import type { SearchSuggestion } from '@/components/v1/molecules/search-bar-with-filters/search-bar-with-filters';

export const SPAN_STATUSES = ['ok', 'error', 'unset'] as const;
export type SpanStatus = (typeof SPAN_STATUSES)[number];

export const SPAN_STATUS_COLORS: Record<SpanStatus, string> = {
  ok: 'bg-green-500',
  error: 'bg-red-500',
  unset: 'bg-slate-500',
};

export interface ParsedTraceQuery {
  search?: string;
  status?: SpanStatus;
  attributes: [string, string][];
  raw: string;
  isValid: boolean;
  errors: string[];
}

export interface TraceAutocompleteSuggestion
  extends SearchSuggestion<'key' | 'value'> {
  type: 'key' | 'value';
  label: string;
  value: string;
  description?: string;
  color?: string;
}

export interface TraceAutocompleteContext {
  attributeKeys: string[];
  attributeValues: Map<string, string[]>;
}
