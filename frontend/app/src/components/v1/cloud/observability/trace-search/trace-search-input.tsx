import {
  getTraceAutocomplete,
  applyTraceSuggestion,
  STATIC_FILTER_KEYS,
} from './autocomplete';
import type {
  TraceAutocompleteSuggestion,
  TraceAutocompleteContext,
} from './types';
import { SearchBarWithFilters } from '@/components/v1/molecules/search-bar-with-filters/search-bar-with-filters';

export function TraceSearchInput({
  value,
  onChange,
  autocompleteContext,
  placeholder = 'Filter spans...',
  className,
}: {
  value: string;
  onChange: (value: string) => void;
  autocompleteContext: TraceAutocompleteContext;
  placeholder?: string;
  className?: string;
}) {
  return (
    <SearchBarWithFilters<TraceAutocompleteSuggestion, TraceAutocompleteContext>
      value={value}
      onChange={onChange}
      onSubmit={onChange}
      getAutocomplete={getTraceAutocomplete}
      applySuggestion={applyTraceSuggestion}
      autocompleteContext={autocompleteContext}
      placeholder={placeholder}
      className={className}
      filterChips={STATIC_FILTER_KEYS.map((f) => ({
        key: f.value,
        label: f.label,
        description: f.description,
      }))}
    />
  );
}
