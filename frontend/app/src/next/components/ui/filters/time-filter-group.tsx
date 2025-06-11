import { TimeFilter, TimeFilterGroup, TogglePause } from './time-filter';

export function TimeFilters() {
  return (
    <TimeFilterGroup className="w-full md:flex-row flex-col justify-end">
      <TimeFilter />
      <TogglePause />
    </TimeFilterGroup>
  );
}
