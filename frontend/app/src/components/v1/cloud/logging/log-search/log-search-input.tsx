import { getAutocompleteContext, getSuggestions } from './autocomplete';
import { parseLogQuery } from './parser';
import { LogSearchInputProps, ParsedLogQuery } from './types';
import {
  Command,
  CommandEmpty,
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
  value,
  onChange,
  onQueryChange,
  placeholder = 'Search logs...',
  showAutocomplete = true,
  className,
}: LogSearchInputProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const blurTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const [isOpen, setIsOpen] = useState(false);
  const [cursorPosition, setCursorPosition] = useState(0);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [parsedQuery, setParsedQuery] = useState<ParsedLogQuery>(() =>
    parseLogQuery(value),
  );

  useEffect(() => {
    const parsed = parseLogQuery(value);
    setParsedQuery(parsed);
    onQueryChange(parsed);
  }, [value, onQueryChange]);

  const autocompleteContext = getAutocompleteContext(value, cursorPosition);
  const suggestions = getSuggestions(autocompleteContext);

  useEffect(() => {
    setSelectedIndex(0);
  }, [suggestions.length, autocompleteContext.mode]);

  const handleInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      onChange(e.target.value);
      setCursorPosition(e.target.selectionStart || 0);
      if (showAutocomplete) {
        setIsOpen(true);
      }
    },
    [onChange, showAutocomplete],
  );

  const handleSuggestionSelect = useCallback(
    (suggestionValue: string) => {
      const beforeCursor = value.slice(0, cursorPosition);
      const afterCursor = value.slice(cursorPosition);

      const lastSpaceIndex = beforeCursor.lastIndexOf(' ');
      const lastColonIndex = beforeCursor.lastIndexOf(':');

      let newValue: string;
      let newCursorPos: number;

      if (lastColonIndex > lastSpaceIndex) {
        const beforeColon = beforeCursor.slice(0, lastColonIndex + 1);
        newValue =
          beforeColon + suggestionValue + ' ' + afterCursor.trimStart();
        newCursorPos = beforeColon.length + suggestionValue.length + 1;
      } else {
        const beforeWord =
          lastSpaceIndex >= 0 ? beforeCursor.slice(0, lastSpaceIndex + 1) : '';
        newValue = beforeWord + suggestionValue + afterCursor.trimStart();
        newCursorPos = beforeWord.length + suggestionValue.length;
      }

      onChange(newValue);
      setCursorPosition(newCursorPos);
      setIsOpen(false);

      setTimeout(() => {
        inputRef.current?.focus();
        inputRef.current?.setSelectionRange(newCursorPos, newCursorPos);
      }, 0);
    },
    [value, cursorPosition, onChange],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      const isDropdownOpen = isOpen && suggestions.length > 0;

      if (e.key === 'Escape') {
        setIsOpen(false);
      } else if (e.key === 'ArrowDown') {
        if (isDropdownOpen) {
          e.preventDefault();
          setSelectedIndex((prev) =>
            prev < suggestions.length - 1 ? prev + 1 : 0,
          );
        } else if (suggestions.length > 0) {
          setIsOpen(true);
        }
      } else if (e.key === 'ArrowUp') {
        if (isDropdownOpen) {
          e.preventDefault();
          setSelectedIndex((prev) =>
            prev > 0 ? prev - 1 : suggestions.length - 1,
          );
        }
      } else if (e.key === 'Enter') {
        if (isDropdownOpen && suggestions[selectedIndex]) {
          e.preventDefault();
          handleSuggestionSelect(suggestions[selectedIndex].value);
        } else {
          e.preventDefault();
          onQueryChange(parsedQuery);
        }
      } else if (
        e.key === 'Tab' &&
        isDropdownOpen &&
        suggestions[selectedIndex]
      ) {
        e.preventDefault();
        handleSuggestionSelect(suggestions[selectedIndex].value);
      }
    },
    [
      isOpen,
      suggestions,
      selectedIndex,
      handleSuggestionSelect,
      onQueryChange,
      parsedQuery,
    ],
  );

  const handleClear = useCallback(() => {
    onChange('');
    setCursorPosition(0);
    inputRef.current?.focus();
  }, [onChange]);

  return (
    <div className={cn('space-y-2', className)}>
      <Popover
        open={isOpen && showAutocomplete && suggestions.length > 0}
        onOpenChange={setIsOpen}
      >
        <PopoverTrigger asChild>
          <div className="relative">
            <MagnifyingGlassIcon className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              ref={inputRef}
              type="text"
              value={value}
              onChange={handleInputChange}
              onKeyDown={handleKeyDown}
              onFocus={() => {
                if (blurTimeoutRef.current) {
                  clearTimeout(blurTimeoutRef.current);
                  blurTimeoutRef.current = null;
                }
                if (suggestions.length > 0) {
                  setIsOpen(true);
                }
              }}
              onBlur={() => {
                blurTimeoutRef.current = setTimeout(() => {
                  setIsOpen(false);
                  blurTimeoutRef.current = null;
                }, 200);
              }}
              onClick={(e) => {
                e.stopPropagation();
                const target = e.target as HTMLInputElement;
                setCursorPosition(target.selectionStart || 0);
              }}
              placeholder={placeholder}
              className={cn(
                'pl-9 pr-8',
                !parsedQuery.isValid &&
                  'border-destructive focus-visible:ring-destructive',
              )}
            />
            {value && (
              <button
                type="button"
                onClick={handleClear}
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
        >
          <Command>
            <CommandList className="max-h-[300px]">
              <CommandEmpty>No suggestions</CommandEmpty>
              {autocompleteContext.mode === 'idle' && (
                <div className="px-3 py-2 text-xs text-muted-foreground border-b">
                  Type to search or select a filter
                </div>
              )}
              <CommandGroup>
                {suggestions.map((suggestion, index) => (
                  <CommandItem
                    key={`${suggestion.value}-${index}`}
                    onSelect={() => handleSuggestionSelect(suggestion.value)}
                    className={cn(
                      'flex items-center justify-between',
                      index === selectedIndex &&
                        'bg-accent text-accent-foreground',
                    )}
                    onMouseEnter={() => setSelectedIndex(index)}
                  >
                    <div className="flex items-center gap-2">
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
        </PopoverContent>
      </Popover>

      {!parsedQuery.isValid && parsedQuery.errors.length > 0 && (
        <div className="text-xs text-destructive">
          {parsedQuery.errors.join(', ')}
        </div>
      )}
    </div>
  );
}
