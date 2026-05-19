# Filter Examples

## Time Range Filter

For date/time picker inputs:

```tsx
import { CustomFilterInputProps } from './search-bar-with-filters';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

function TimeRangeFilterInput({
  filterKey: _filterKey,
  currentValue,
  onComplete,
  onCancel,
}: CustomFilterInputProps) {
  const [start, setStart] = useState('');
  const [end, setEnd] = useState('');

  return (
    <div className="p-4 space-y-3">
      <div className="text-sm font-medium">Select Time Range</div>
      <Input
        type="datetime-local"
        value={start}
        onChange={(e) => setStart(e.target.value)}
      />
      <Input
        type="datetime-local"
        value={end}
        onChange={(e) => setEnd(e.target.value)}
      />
      <div className="flex gap-2">
        <Button variant="outline" size="sm" onClick={onCancel}>
          Cancel
        </Button>
        <Button
          size="sm"
          onClick={() => onComplete(`${start},${end}`)}
          disabled={!start || !end}
        >
          Apply
        </Button>
      </div>
    </div>
  );
}

// Complete usage with SearchBarWithFilters:
function MySearchComponent() {
  const [query, setQuery] = useState('');

  const getAutocomplete = (q: string) => {
    const lastWord = q.split(' ').pop() || '';

    // Show time range picker when "time:" is typed
    if (lastWord.startsWith('time:')) {
      return {
        suggestions: [
          {
            type: 'custom-input',
            label: 'Select time range',
            value: 'time:',
            customInput: TimeRangeFilterInput,
          },
        ],
      };
    }

    // Return other suggestions...
    return { suggestions: [] };
  };

  const applySuggestion = (q: string, suggestion: any) => {
    // Apply the time range value
    return q + suggestion.value;
  };

  return (
    <SearchBarWithFilters
      value={query}
      onChange={setQuery}
      getAutocomplete={getAutocomplete}
      applySuggestion={applySuggestion}
      filterChips={[
        {
          key: 'time:',
          label: 'Time Range',
          customInput: TimeRangeFilterInput,
          requiresCustomInput: true,
        },
      ]}
    />
  );
}
```

## Multi-Select Filter

For selecting multiple values:

```tsx
import { Checkbox } from '@/components/ui/checkbox';

function MultiSelectFilterInput({
  currentValue,
  onComplete,
  onCancel,
}: CustomFilterInputProps) {
  const options = [
    { value: 'active', label: 'Active', color: 'bg-green-500' },
    { value: 'pending', label: 'Pending', color: 'bg-yellow-500' },
    { value: 'failed', label: 'Failed', color: 'bg-red-500' },
  ];

  const [selected, setSelected] = useState<string[]>(
    currentValue ? currentValue.split(',') : [],
  );

  return (
    <div className="p-4 space-y-3 min-w-[250px]">
      <div className="text-sm font-medium">Select Statuses</div>
      <div className="space-y-2">
        {options.map((option) => (
          <div key={option.value} className="flex items-center gap-2">
            <Checkbox
              checked={selected.includes(option.value)}
              onCheckedChange={() => {
                setSelected((prev) =>
                  prev.includes(option.value)
                    ? prev.filter((v) => v !== option.value)
                    : [...prev, option.value],
                );
              }}
            />
            <div className={`h-2 w-2 rounded-full ${option.color}`} />
            <span className="text-sm">{option.label}</span>
          </div>
        ))}
      </div>
      <div className="flex justify-end gap-2">
        <Button variant="outline" size="sm" onClick={onCancel}>
          Cancel
        </Button>
        <Button
          size="sm"
          onClick={() => onComplete(selected.join(','))}
          disabled={selected.length === 0}
        >
          Apply ({selected.length})
        </Button>
      </div>
    </div>
  );
}

// Complete usage with SearchBarWithFilters:
function MySearchComponent() {
  const [query, setQuery] = useState('');

  const getAutocomplete = (q: string) => {
    const lastWord = q.split(' ').pop() || '';

    // Show multi-select when "status:" is typed
    if (lastWord.startsWith('status:')) {
      return {
        suggestions: [
          {
            type: 'multi-select',
            label: 'Select statuses',
            value: 'status:',
            customInput: MultiSelectFilterInput,
          },
        ],
      };
    }

    // Return other suggestions...
    return { suggestions: [] };
  };

  const applySuggestion = (q: string, suggestion: any) => {
    // Apply the selected statuses
    return q + suggestion.value;
  };

  return (
    <SearchBarWithFilters
      value={query}
      onChange={setQuery}
      getAutocomplete={getAutocomplete}
      applySuggestion={applySuggestion}
      filterChips={[
        {
          key: 'status:',
          label: 'Status',
          customInput: MultiSelectFilterInput,
          requiresCustomInput: true,
        },
      ]}
    />
  );
}
```

## Numeric Range Filter

For port ranges, counts, etc:

```tsx
function RangeFilterInput({
  filterKey,
  currentValue,
  onComplete,
  onCancel,
}: CustomFilterInputProps) {
  const [min, setMin] = useState(0);
  const [max, setMax] = useState(0);

  return (
    <div className="p-4 space-y-3">
      <div className="text-sm font-medium">
        Select {filterKey.replace(':', '')} Range
      </div>
      <div className="flex items-center gap-2">
        <input
          type="number"
          value={min || ''}
          onChange={(e) => setMin(Number(e.target.value))}
          placeholder="Min"
          className="flex-1 px-3 py-2 text-sm border rounded-md"
        />
        <span className="text-muted-foreground">to</span>
        <input
          type="number"
          value={max || ''}
          onChange={(e) => setMax(Number(e.target.value))}
          placeholder="Max"
          className="flex-1 px-3 py-2 text-sm border rounded-md"
        />
      </div>
      <div className="flex justify-end gap-2">
        <Button variant="outline" size="sm" onClick={onCancel}>
          Cancel
        </Button>
        <Button
          size="sm"
          onClick={() => onComplete(`${min}-${max}`)}
          disabled={!min || !max || min > max}
        >
          Apply
        </Button>
      </div>
    </div>
  );
}

// Complete usage with SearchBarWithFilters:
function MySearchComponent() {
  const [query, setQuery] = useState('');

  const getAutocomplete = (q: string) => {
    const lastWord = q.split(' ').pop() || '';

    // Show range input when "port:" is typed
    if (lastWord.startsWith('port:')) {
      return {
        suggestions: [
          {
            type: 'range',
            label: 'Enter port range',
            value: 'port:',
            customInput: RangeFilterInput,
          },
        ],
      };
    }

    // Return other suggestions...
    return { suggestions: [] };
  };

  const applySuggestion = (q: string, suggestion: any) => {
    // Apply the port range
    return q + suggestion.value;
  };

  return (
    <SearchBarWithFilters
      value={query}
      onChange={setQuery}
      getAutocomplete={getAutocomplete}
      applySuggestion={applySuggestion}
      filterChips={[
        {
          key: 'port:',
          label: 'Port Range',
          customInput: RangeFilterInput,
          requiresCustomInput: true,
        },
      ]}
    />
  );
}
```
