import { getAutocomplete, applySuggestion } from './autocomplete';
import { useLogsContext } from './use-logs';
import { Button } from '@/components/v1/ui/button';
import {
  Command,
  CommandGroup,
  CommandItem,
  CommandList,
} from '@/components/v1/ui/command';
import { Input } from '@/components/v1/ui/input';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import { cn } from '@/lib/utils';
import { MagnifyingGlassIcon, Cross2Icon } from '@radix-ui/react-icons';
import React, { useState, useRef, useCallback, useEffect } from 'react';

export function LogSearchInput({
  placeholder = 'Search logs...',
  className,
}: {
  placeholder?: string;
  className?: string;
}) {
  const { queryString, setQueryString, availableAttempts } = useLogsContext();
  const inputRef = useRef<HTMLInputElement>(null);
  const blurTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const [isOpen, setIsOpen] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState<number>();
  const [localValue, setLocalValue] = useState(queryString);

  useEffect(() => {
    setLocalValue(queryString);
  }, [queryString]);

  const { suggestions } = getAutocomplete(localValue, availableAttempts);

  const submitSearch = useCallback(() => {
    setQueryString(localValue);
  }, [localValue, setQueryString]);

  const handleFilterChipClick = useCallback(
    (filterKey: string) => {
      const newValue = localValue ? `${localValue} ${filterKey}` : filterKey;
      setLocalValue(newValue);
      setIsOpen(true);
      setSelectedIndex(undefined);
      setTimeout(() => {
        const input = inputRef.current;
        if (input) {
          input.focus();
          input.setSelectionRange(newValue.length, newValue.length);
        }
      }, 0);
    },
    [localValue],
  );

  const handleSelect = useCallback(
    (index: number) => {
      const suggestion = suggestions[index];
      if (suggestion) {
        const newValue = applySuggestion(localValue, suggestion);
        setLocalValue(newValue);
        setQueryString(newValue);
        setIsOpen(false);
        setTimeout(() => {
          const input = inputRef.current;
          if (input) {
            input.focus();
            input.setSelectionRange(newValue.length, newValue.length);
          }
        }, 0);
      }
    },
    [localValue, suggestions, setQueryString],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        if (isOpen && suggestions.length > 0 && selectedIndex !== undefined) {
          handleSelect(selectedIndex);
        } else {
          submitSearch();
        }
        setIsOpen(false);
        return;
      }

      if (!isOpen || suggestions.length === 0) {
        return;
      }

      if (e.key === 'Escape') {
        setIsOpen(false);
        setSelectedIndex(undefined);
      } else if (e.key === 'ArrowDown') {
        e.preventDefault();
        setSelectedIndex((i) => {
          if (i === undefined) {
            return 0;
          }
          return i < suggestions.length - 1 ? i + 1 : 0;
        });
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setSelectedIndex((i) => {
          if (i === undefined) {
            return suggestions.length - 1;
          }
          return i > 0 ? i - 1 : suggestions.length - 1;
        });
      } else if (e.key === 'Tab') {
        if (selectedIndex !== undefined) {
          e.preventDefault();
          handleSelect(selectedIndex);
        }
      }
    },
    [isOpen, suggestions.length, selectedIndex, handleSelect, submitSearch],
  );

  return (
    <div className={cn('space-y-2', className)}>
      <Popover open={isOpen} modal={false}>
        <PopoverTrigger asChild>
          <div className="relative">
            <MagnifyingGlassIcon className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              ref={inputRef}
              type="text"
              value={localValue}
              onChange={(e) => {
                setLocalValue(e.target.value);
                setIsOpen(true);
                setSelectedIndex(undefined);
              }}
              onKeyDown={handleKeyDown}
              onFocus={() => {
                if (blurTimeoutRef.current) {
                  clearTimeout(blurTimeoutRef.current);
                  blurTimeoutRef.current = null;
                }
                setIsOpen(true);
                setSelectedIndex(undefined);
              }}
              onBlur={() => {
                blurTimeoutRef.current = setTimeout(() => {
                  setIsOpen(false);
                  blurTimeoutRef.current = null;
                }, 200);
              }}
              placeholder={placeholder}
              className="pl-9 pr-8"
            />
            {localValue && (
              <button
                type="button"
                onClick={() => {
                  setLocalValue('');
                  setQueryString('');
                  inputRef.current?.focus();
                }}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                <Cross2Icon className="h-4 w-4" />
              </button>
            )}
          </div>
        </PopoverTrigger>
        <PopoverContent
          className="w-[var(--radix-popover-trigger-width)] min-w-[320px] p-0"
          align="start"
          onOpenAutoFocus={(e) => e.preventDefault()}
          onCloseAutoFocus={(e) => e.preventDefault()}
        >
          {suggestions.length > 0 && (
            <Command
              value={
                selectedIndex !== undefined
                  ? suggestions[selectedIndex]?.value
                  : ''
              }
            >
              <CommandList className="max-h-[300px]">
                <CommandGroup>
                  {suggestions.map((suggestion, index) => (
                    <CommandItem
                      key={suggestion.value}
                      value={suggestion.value}
                      onSelect={() => handleSelect(index)}
                      className={cn(
                        'flex items-center justify-between',
                        selectedIndex !== undefined &&
                          index === selectedIndex &&
                          'bg-accent text-accent-foreground',
                      )}
                      onMouseEnter={() => setSelectedIndex(index)}
                    >
                      <div className="flex items-center gap-2">
                        {suggestion.color && (
                          <div
                            className={cn(
                              suggestion.color,
                              'h-[6px] w-[6px] rounded-full',
                            )}
                          />
                        )}
                        {suggestion.type === 'key' ? (
                          <code className="px-1.5 py-0.5 bg-muted rounded text-xs font-mono">
                            {suggestion.label}:
                          </code>
                        ) : (
                          <span className="font-mono text-sm">
                            {suggestion.label}
                          </span>
                        )}
                      </div>
                      {suggestion.description && (
                        <span className="text-xs text-muted-foreground truncate ml-2">
                          {suggestion.description}
                        </span>
                      )}
                    </CommandItem>
                  ))}
                </CommandGroup>
              </CommandList>
            </Command>
          )}
          <div
            className={cn(
              'flex items-center gap-2 px-3 py-2 text-xs',
              suggestions.length > 0 && 'border-t',
            )}
          >
            <span className="text-muted-foreground">Available filters:</span>
            <Button
              variant="outline"
              size="xs"
              className="h-auto px-2 py-0.5 text-xs"
              onClick={() => handleFilterChipClick('level:')}
            >
              Level
            </Button>
            <Button
              variant="outline"
              size="xs"
              className="h-auto px-2 py-0.5 text-xs"
              onClick={() => handleFilterChipClick('attempt:')}
            >
              Attempt
            </Button>
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
