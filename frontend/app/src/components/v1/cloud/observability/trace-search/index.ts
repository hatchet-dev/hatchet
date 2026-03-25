export { filterSpanTrees } from './filter';
export type { FilteredSpanTree } from './filter';
export { parseTraceQuery } from './parser';
export { TraceSearchInput } from './trace-search-input';
export { getTraceAutocomplete, applyTraceSuggestion } from './autocomplete';
export type { AutocompleteMode } from './autocomplete';
export type {
  TraceAutocompleteSuggestion,
  SpanStatus,
  ParsedTraceQuery,
  TraceAutocompleteContext,
} from './types';
