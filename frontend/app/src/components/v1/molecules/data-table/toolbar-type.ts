// Kept in its own module so components like Combobox can use it without
// importing the data table toolbar (and the pages it depends on).
export enum ToolbarType {
  Checkbox = 'checkbox',
  Radio = 'radio',
  KeyValue = 'key-value',
  Array = 'array',
  Switch = 'switch',
  TimeRange = 'time-range',
  Search = 'search',
}
