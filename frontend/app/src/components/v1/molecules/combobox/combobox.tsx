import { ToolbarType } from '../data-table/data-table-toolbar';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/v1/ui/command';
import { Input } from '@/components/v1/ui/input';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import { Separator } from '@/components/v1/ui/separator';
import { cn } from '@/lib/utils';
import { zodResolver } from '@hookform/resolvers/zod';
import { CheckIcon, PlusCircledIcon } from '@radix-ui/react-icons';
import * as React from 'react';
import { useForm } from 'react-hook-form';
import { BiX } from 'react-icons/bi';
import { z } from 'zod';

const keyValuePairSchema = z.object({
  key: z.string().min(1, 'Key is required'),
  value: z.string().min(1, 'Value is required'),
});

const arrayInputSchema = z.object({
  values: z.string().min(1, 'At least one value is required'),
});

type KeyValuePair = z.infer<typeof keyValuePairSchema>;
type ArrayInput = z.infer<typeof arrayInputSchema>;

export function Combobox({
  values = [],
  title,
  icon,
  type = ToolbarType.Checkbox,
  options,
  setValues,
  onSearchChange,
  searchValue,
  emptyMessage,
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
  onSearchChange?: (value: string) => void;
  searchValue?: string;
  emptyMessage?: string;
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
          {icon || <PlusCircledIcon className="mr-2 size-4" />}
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
                      className="flex items-center space-x-1 rounded-sm px-1 font-normal"
                    >
                      {options?.find(({ value }) => value == option)?.label ||
                        option}
                      <Button
                        variant="ghost"
                        size="xs"
                        className="ml-2"
                        onClick={() => remove(option)}
                      >
                        <BiX className="size-3" />
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
                  className="mb-2 mr-2 flex items-center space-x-1 rounded-sm px-1 pl-2 font-normal"
                >
                  <span className="grow">{filter}</span>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="ml-2 shrink-0"
                    onClick={() => remove(filter)}
                  >
                    <BiX className="size-4" />
                  </Button>
                </Badge>
              ))}
            </div>
            <form onSubmit={handleSubmit(submit)}>
              {type === ToolbarType.KeyValue ? (
                <div className="mb-2 flex items-center space-x-2">
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
              <Button type="submit" fullWidth size="sm">
                Add {title} Filter
              </Button>
              {values.length > 0 && (
                <Button
                  onClick={() => setValues([])}
                  className="mt-2"
                  size="sm"
                  variant={'ghost'}
                  fullWidth
                >
                  Reset
                </Button>
              )}
            </form>
          </div>
        )}

        {[ToolbarType.Checkbox, ToolbarType.Radio].includes(type) && (
          <Command shouldFilter={!onSearchChange}>
            <CommandInput
              placeholder={title}
              value={searchValue}
              onValueChange={onSearchChange}
            />
            <CommandList>
              <CommandEmpty className="py-2 text-center text-sm text-muted-foreground">
                {emptyMessage || 'No results found.'}
              </CommandEmpty>
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
                          if (type === ToolbarType.Radio) {
                            setValues([option.value]);
                            return;
                          } else {
                            values.push(option.value);
                          }
                        }
                        setValues(values);
                      }}
                    >
                      <div
                        className={cn(
                          'mr-2 flex size-4 items-center justify-center rounded-sm border border-primary',
                          isSelected
                            ? 'bg-primary text-primary-foreground'
                            : 'opacity-50 [&_svg]:invisible',
                        )}
                      >
                        <CheckIcon className={cn('size-4')} />
                      </div>
                      {option.icon && (
                        <option.icon className="mr-2 size-4 text-gray-700 dark:text-gray-300" />
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
