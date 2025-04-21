import * as React from 'react';
import { cn } from '@/next/lib/utils';
import { Input } from '@/next/components/ui/input';
import { useFilters } from '@/next/hooks/use-filters';
import { Badge } from '@/next/components/ui/badge';
import { Check } from 'lucide-react';
import { Button } from '@/next/components/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
} from '@/next/components/ui/command';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/next/components/ui/popover';
import useDefinitions from '@/next/hooks/use-definitions';
import { useMemo } from 'react';
import { PlusCircledIcon } from '@radix-ui/react-icons';

interface FiltersProps {
  children?: React.ReactNode;
  className?: string;
}

interface FilterBuilderProps<T> {
  name: keyof T;
  placeholder?: string;
  className?: string;
}

export function FilterGroup({ className, children, ...props }: FiltersProps) {
  return (
    <div
      role="filters"
      aria-label="filters"
      className={cn('flex w-full items-center gap-2 md:gap-6', className)}
      {...props}
    >
      {children}
    </div>
  );
}

interface TextFilterProps<T> extends FilterBuilderProps<T> {
  value?: string;
}

export function FilterText<T>({
  name,
  placeholder,
  className,
}: TextFilterProps<T>) {
  const { filters, setFilter } = useFilters<T>();
  const value = filters[name];

  return (
    <Input
      type="text"
      value={value as string}
      onChange={(e) =>
        setFilter(
          name,
          e.target.value
            ? (e.target.value as T[keyof T])
            : (undefined as T[keyof T]),
        )
      }
      placeholder={placeholder}
      className={cn('flex-grow h-8 rounded-md px-3 text-xs', className)}
    />
  );
}

type ArrayElement<T> = T extends (infer U)[] ? U : T;

interface MultiSelectFilterProps<T, A, Key extends keyof T = keyof T>
  extends FilterBuilderProps<T> {
  multi?: boolean;
  name: Key;
  only?: boolean;
  options: {
    label: React.ReactNode;
    value: ArrayElement<A>;
    text?: string;
  }[];
  value?: A;
}

export function FilterSelect<T, A>({
  name,
  options,
  placeholder,
  multi = false,
  only = false,
}: MultiSelectFilterProps<T, A>) {
  const { filters, setFilter } = useFilters<T>();
  const value = filters[name] as Array<ArrayElement<A>> | undefined;
  const [open, setOpen] = React.useState(false);

  const handleSelect = (optionValue: ArrayElement<A>) => {
    if (multi && value?.includes(optionValue)) {
      setFilter(
        name,
        value.filter((v) => v !== optionValue) as unknown as T[keyof T],
      );
    } else if (!multi && value === optionValue) {
      setFilter(name, optionValue as T[keyof T]);
    } else {
      setFilter(
        name,
        multi
          ? ([...(value || []), optionValue] as unknown as T[keyof T])
          : (optionValue as T[keyof T]),
      );
    }
  };

  const selectedOptions = multi
    ? value?.map((v) => options.find((o) => o.value === v)).filter(Boolean) ||
      []
    : value
      ? [options.find((o) => o.value === value)]
      : [];

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          role="combobox"
          aria-expanded={open}
          className="w-fit min-w-fit gap-2"
        >
          <PlusCircledIcon className="h-4 w-4" />
          <div className="flex flex-wrap gap-1">
            {selectedOptions.length > 0 ? (
              selectedOptions.map((option, index) => (
                <Badge key={index} variant="secondary" className="mr-1">
                  {option?.label}
                </Badge>
              ))
            ) : (
              <span className="text-foreground">{placeholder}</span>
            )}
          </div>
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-full p-0" side="bottom" align="start">
        <Command>
          <CommandInput placeholder={placeholder} />
          <CommandEmpty>No results found.</CommandEmpty>
          <CommandGroup>
            {options.map((option, index) => (
              <CommandItem
                key={index}
                onSelect={() => {
                  handleSelect(option.value);
                }}
                className="group"
              >
                <div className="flex items-center space-x-2 justify-between">
                  <div
                    className={cn(
                      'flex h-4 w-4 items-center justify-center rounded-sm border border-primary',
                      (multi && value?.includes(option.value)) ||
                        (!multi && value === option.value)
                        ? 'bg-primary text-primary-foreground'
                        : 'opacity-50 [&_svg]:invisible',
                    )}
                  >
                    {multi && <Check className="h-4 w-4" />}
                  </div>
                  <span>{option.text || option.label}</span>
                </div>

                {multi && only && (
                  <Badge
                    variant="outline"
                    className="ml-auto opacity-0 group-hover:opacity-100"
                    onClick={(e) => {
                      e.stopPropagation();
                      setFilter(name, [option.value] as T[keyof T]);
                    }}
                  >
                    Only
                  </Badge>
                )}
              </CommandItem>
            ))}
          </CommandGroup>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

interface FilterTaskSelectProps<T> extends FilterBuilderProps<T> {
  multi?: boolean;
  only?: boolean;
  value?: string;
}

export function FilterTaskSelect<T>({ ...props }: FilterTaskSelectProps<T>) {
  const { data: options = [] } = useDefinitions();
  return (
    <FilterSelect<T, string>
      options={options.map((o) => ({
        label: o.name,
        value: o.metadata.id,
      }))}
      {...props}
    />
  );
}

interface FilterKeyValueProps<T> extends FilterBuilderProps<T> {
  options?: {
    label: string;
    value: string;
  }[];
}

export function FilterKeyValue<T>({
  name,
  options = [],
  placeholder = 'Add filter',
}: FilterKeyValueProps<T>) {
  const { filters, setFilter } = useFilters<T>();
  const [open, setOpen] = React.useState(false);
  const [key, setKey] = React.useState<string>('');
  const [value, setValue] = React.useState<string>('');

  const currentFilters = useMemo(
    () => (filters[name] as string[]) || [],
    [filters, name],
  );

  const handleAddFilter = () => {
    if (key && value) {
      const newFilter = `${key}:${value}`;
      // Check if this exact key-value pair already exists
      if (currentFilters.includes(newFilter)) {
        return; // Don't add duplicate
      }
      setFilter(name, [...currentFilters, newFilter] as T[keyof T]);
      setKey('');
      setValue('');
      setOpen(false);
    }
  };

  const handleRemoveFilter = (index: number) => {
    setFilter(name, currentFilters.filter((_, i) => i !== index) as T[keyof T]);
  };

  // Get unique keys from current filters
  const existingKeys = useMemo(() => {
    return new Set(currentFilters.map((filter) => filter.split(':')[0]));
  }, [currentFilters]);

  // Filter and deduplicate options
  const filteredOptions = useMemo(() => {
    const seen = new Set<string>();
    return options
      .filter(
        (option) =>
          option.label.toLowerCase().includes(key.toLowerCase()) &&
          !existingKeys.has(option.label) &&
          option.label !== key,
      )
      .filter((option) => {
        if (seen.has(option.label)) {
          return false;
        }
        seen.add(option.label);
        return true;
      });
  }, [options, key, existingKeys]);

  return (
    <div className="flex flex-col gap-2">
      {currentFilters.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {currentFilters.map((filter, index) => {
            const [key, value] = filter.split(':');
            return (
              <Badge
                key={index}
                variant="secondary"
                className="flex items-center gap-1"
              >
                <span>{key === value ? value : `${key}: ${value}`}</span>
                <button
                  onClick={() => handleRemoveFilter(index)}
                  className="ml-1 hover:brightness-75"
                >
                  ×
                </button>
              </Badge>
            );
          })}
        </div>
      )}
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            className="w-full justify-start text-foreground gap-2"
          >
            <PlusCircledIcon className="h-4 w-4" />
            {placeholder}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-80">
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Input
                placeholder="Enter key"
                value={key}
                onChange={(e) => setKey(e.target.value)}
              />
              {filteredOptions.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  {filteredOptions.map((option) => (
                    <Badge
                      key={option.value}
                      variant="outline"
                      className="cursor-pointer"
                      onClick={() => setKey(option.label)}
                    >
                      {option.label}
                    </Badge>
                  ))}
                </div>
              )}
            </div>
            <Input
              placeholder="Enter value"
              value={value}
              onChange={(e) => setValue(e.target.value)}
            />
            <Button
              onClick={handleAddFilter}
              disabled={
                !key || !value || currentFilters.includes(`${key}:${value}`)
              }
            >
              Add Filter
            </Button>
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
