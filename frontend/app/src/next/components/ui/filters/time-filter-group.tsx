import { RunsFilters } from '@/next/hooks/use-runs';
import { ClearFiltersButton, FilterGroup } from './filters';
import { TimeFilter } from './time-filter';

export function TimeFilterGroup() {
  return (
    <FilterGroup>
      <TimeFilter<RunsFilters>
        startField="createdAfter"
        endField="createdBefore"
      />
      <ClearFiltersButton />
    </FilterGroup>
  );
}
