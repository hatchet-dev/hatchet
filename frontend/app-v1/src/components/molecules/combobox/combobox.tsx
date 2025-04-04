import * as React from 'react';
import { CheckIcon, CircleIcon, PlusCircledIcon } from '@radix-ui/react-icons';
import { Column } from '@tanstack/react-table';

import { cn } from '@/lib/utils';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { Separator } from '@/components/ui/separator';
import { ToolbarType } from '../data-table/data-table-toolbar';
import { Input } from '@/components/ui/input';
import { BiX } from 'react-icons/bi';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';

interface DataTableFacetedFilterProps<TData, TValue> {
  column?: Column<TData, TValue>;
  title?: string;
  type?: ToolbarType;
  options?: {
    label: string;
    value: string;
    icon?: React.ComponentType<{ className?: string }>;
  }[];
}

const keyValuePairSchema = z.object({
  key: z.string().min(1, 'Key is required'),
  value: z.string().min(1, 'Value is required'),
});

const arrayInputSchema = z.object({
  values: z.string().min(1, 'At least one value is required'),
});

type KeyValuePair = z.infer<typeof keyValuePairSchema>;
type ArrayInput = z.infer<typeof arrayInputSchema>;

export function DataTableFacetedFilter<TData, TValue>({
  column,
  title,
  type = ToolbarType.Checkbox,
  options,
}: DataTableFacetedFilterProps<TData, TValue>) {
  return (
    <Combobox
      values={column?.getFilterValue() as string[]}
      title={title}
      type={type}
      options={options}
      setValues={(values) => column?.setFilterValue(values)}
    />
  );
}

export function Combobox({
  values = [],
  title,
  icon,
  type = ToolbarType.Checkbox,
  options,
  setValues,
}: {
  values?: string[];
  icon?: JSX.Element;
  title?: string;
  type?: ToolbarType;
  options?: {
    label: string;
    value: string;
    icon?: React.ComponentType<{ className?: string }>;
  }[];
  setValues: (selectedValues: string[]) => void;
}) {
  const { register, handleSubmit, reset } = useForm<KeyValuePair | ArrayInput>({
    resolver: zodResolver(
      type === ToolbarType.KeyValue ? keyValuePairSchema : arrayInputSchema,
    ),
    defaultValues:
      type === ToolbarType.KeyValue ? { key: '', value: '' } : { values: '' },
  });

  const submit = (data: KeyValuePair | ArrayInput) => {
    if ('key' in data) {
      values.push(`${data.key}:${data.value}`);
    } else {
      data.values.split(',').forEach((value) => values.push(value.trim()));
    }
    setValues(values);
    reset();
  };

  const remove = (filter: string) => {
    const index = values.indexOf(filter);
    if (index > -1) {
      values.splice(index, 1);
    }
    setValues(values);
  };

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="h-8 border-dashed">
          {icon || <PlusCircledIcon className="mr-2 h-4 w-4" />}
          {title}
          {values.length > 0 && (
            <>
              <Separator orientation="vertical" className="mx-2 h-4" />
              <Badge
                variant="secondary"
                className="rounded-sm px-1 font-normal lg:hidden"
              >
                {type == ToolbarType.Radio
                  ? // get the label of the value
                    options?.find(({ value }) => value == values[0])?.label ||
                    values[0]
                  : values.length}
              </Badge>
              <div className="hidden space-x-1 lg:flex">
                {values.length > 2 ? (
                  <Badge
                    variant="secondary"
                    className="rounded-sm px-1 font-normal"
                  >
                    {values.length} selected
                  </Badge>
                ) : (
                  values.map((option, index) => (
                    <Badge
                      key={index}
                      variant="secondary"
                      className="rounded-sm px-1 font-normal flex items-center space-x-1"
                    >
                      {options?.find(({ value }) => value == option)?.label ||
                        option}
                      <Button
                        variant="ghost"
                        size="xs"
                        className="ml-2"
                        onClick={() => remove(option)}
                      >
                        <BiX className="h-3 w-3" />
                      </Button>
                    </Badge>
                  ))
                )}
              </div>
            </>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[300px] p-2" align="start">
        {[ToolbarType.Array, ToolbarType.KeyValue].includes(type) && (
          <div>
            <div className="">
              {values.map((filter, index) => (
                <Badge
                  key={index}
                  variant="secondary"
                  className="mr-2 mb-2 rounded-sm px-1 font-normal flex items-center space-x-1 pl-2"
                >
                  <span className="grow">{filter}</span>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="ml-2 shrink-0"
                    onClick={() => remove(filter)}
                  >
                    <BiX className="h-4 w-4" />
                  </Button>
                </Badge>
              ))}
            </div>
            <form onSubmit={handleSubmit(submit)}>
              {type === ToolbarType.KeyValue ? (
                <div className="flex items-center space-x-2 mb-2">
                  <Input
                    type="text"
                    placeholder="Key"
                    {...register('key')}
                    className="flex-1"
                  />
                  <Input
                    type="text"
                    placeholder="Value"
                    {...register('value')}
                    className="flex-1"
                  />
                </div>
              ) : (
                <div className="mb-2">
                  <Input
                    type="text"
                    placeholder="Enter values (comma-separated)"
                    {...register('values')}
                    className="w-full"
                  />
                </div>
              )}
              <Button type="submit" className="w-full" size="sm">
                Add {title} Filter
              </Button>
              {values.length > 0 && (
                <Button
                  onClick={() => setValues([])}
                  className="w-full mt-2"
                  size="sm"
                  variant={'ghost'}
                >
                  Reset
                </Button>
              )}
            </form>
          </div>
        )}

        {[ToolbarType.Checkbox, ToolbarType.Radio].includes(type) && (
          <Command>
            <CommandInput placeholder={title} />
            <CommandList>
              <CommandEmpty>No results found.</CommandEmpty>
              <CommandGroup>
                {options?.map((option) => {
                  const isSelected = values.indexOf(option.value) != -1;
                  return (
                    <CommandItem
                      key={option.value}
                      onSelect={() => {
                        if (isSelected) {
                          values.splice(values.indexOf(option.value), 1);
                        } else {
                          if (type == 'radio') {
                            values = [];
                          }
                          values.push(option.value);
                        }
                        setValues(values);
                      }}
                    >
                      <div
                        className={cn(
                          'mr-2 flex h-4 w-4 items-center justify-center rounded-sm border border-primary',
                          isSelected
                            ? 'bg-primary text-primary-foreground'
                            : 'opacity-50 [&_svg]:invisible',
                        )}
                      >
                        {type === 'checkbox' ? (
                          <CheckIcon className={cn('h-4 w-4')} />
                        ) : (
                          <CircleIcon className={cn('h-4 w-4')} />
                        )}
                      </div>
                      {option.icon && (
                        <option.icon className="mr-2 h-4 w-4 text-gray-700 dark:text-gray-300" />
                      )}
                      <span>{option.label}</span>
                    </CommandItem>
                  );
                })}
              </CommandGroup>
              {values.length > 0 && (
                <>
                  <CommandSeparator />
                  <CommandGroup>
                    <CommandItem
                      onSelect={() => setValues([])}
                      className="justify-center text-center"
                    >
                      Reset
                    </CommandItem>
                  </CommandGroup>
                </>
              )}
            </CommandList>
          </Command>
        )}
      </PopoverContent>
    </Popover>
  );
}
