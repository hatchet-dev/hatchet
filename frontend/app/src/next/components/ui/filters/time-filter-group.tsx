import { TimeFilter, TimeFilterGroup, TogglePause } from './time-filter';

export function TimeFilters() {
  return (
    <TimeFilterGroup className="w-full justify-end">
      <TimeFilter />
      <TogglePause />
    </TimeFilterGroup>
  );
}
