import { getAutocomplete, applySuggestion } from './autocomplete';
import type { LogAutocompleteContext } from './autocomplete';
import type { AutocompleteSuggestion } from './types';
import { useLogsContext } from './use-logs';
import { SearchBarWithFilters } from '@/components/v1/molecules/search-bar-with-filters/search-bar-with-filters';
import { useMemo } from 'react';

export function LogSearchInput({
  placeholder = 'Search logs...',
  className,
}: {
  placeholder?: string;
  className?: string;
}) {
  const { queryString, setQueryString, availableAttempts } = useLogsContext();

  const autocompleteContext = useMemo<LogAutocompleteContext>(
    () => ({ availableAttempts }),
    [availableAttempts],
  );

  return (
    <SearchBarWithFilters<AutocompleteSuggestion, LogAutocompleteContext>
      value={queryString}
      onChange={setQueryString}
      onSubmit={setQueryString}
      getAutocomplete={getAutocomplete}
      applySuggestion={applySuggestion}
      autocompleteContext={autocompleteContext}
      placeholder={placeholder}
      className={className}
      filterChips={[
        { key: 'level:', label: 'Level', description: 'Filter by log level' },
        {
          key: 'attempt:',
          label: 'Attempt',
          description: 'Filter by attempt number',
        },
      ]}
    />
  );
}
