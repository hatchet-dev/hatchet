import { TimeFilter, TimeFilterGroup, TogglePause } from './time-filter';

export function TimeFilters() {
  return (
    <TimeFilterGroup className="w-full justify-end mb-4">
      <TimeFilter />
      <TogglePause />
    </TimeFilterGroup>
  );
}
