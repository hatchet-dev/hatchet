import {
  getAutocomplete,
  applySuggestion,
  STATIC_FILTER_KEYS,
} from './autocomplete';
import type { AutocompleteSuggestion } from './types';
import { useLogsContext } from './use-logs';
import { SearchBarWithFilters } from '@/components/v1/molecules/search-bar-with-filters/search-bar-with-filters';

export function LogSearchInput({
  placeholder = 'Search logs...',
  className,
}: {
  placeholder?: string;
  className?: string;
}) {
  const { queryString, setQueryString, availableAttempts } = useLogsContext();

  return (
    <SearchBarWithFilters<AutocompleteSuggestion, number[]>
      value={queryString}
      onChange={setQueryString}
      onSubmit={setQueryString}
      getAutocomplete={getAutocomplete}
      applySuggestion={applySuggestion}
      autocompleteContext={availableAttempts}
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
