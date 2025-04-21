import { ClearFiltersButton, FilterGroup } from './filters';
import { TimeFilter } from './time-filter';

export function TimeFilterGroup() {
  return (
    <FilterGroup>
      <TimeFilter />
      <ClearFiltersButton />
    </FilterGroup>
  );
}
