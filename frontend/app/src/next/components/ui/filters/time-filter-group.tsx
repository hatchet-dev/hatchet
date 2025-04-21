import { TimeFilter, TimeFilterGroup, TogglePause } from './time-filter';

export function TimeFilters() {
  return (
    <TimeFilterGroup>
      <TimeFilter />
      <TogglePause />
    </TimeFilterGroup>
  );
}
