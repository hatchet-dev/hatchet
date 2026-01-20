import React, { useState, useRef, useCallback, useEffect } from 'react';
import { Input } from '@/components/v1/ui/input';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandList,
} from '@/components/v1/ui/command';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
  DropdownMenuLabel,
} from '@/components/v1/ui/dropdown-menu';
import { cn } from '@/lib/utils';
import {
  MagnifyingGlassIcon,
  Cross2Icon,
  ChevronDownIcon,
  ClockIcon,
  LayersIcon,
  InfoCircledIcon,
} from '@radix-ui/react-icons';

import { LogSearchInputProps, ParsedLogQuery, QueryToken } from './types';
import { parseLogQuery } from './parser';
import { getAutocompleteContext, getSuggestions } from './autocomplete';

const LOG_LEVELS = ['debug', 'info', 'warn', 'error', 'fatal'] as const;

const TIME_PRESETS = [
  { label: 'Last 15 minutes', value: '15m' },
  { label: 'Last hour', value: '1h' },
  { label: 'Last 6 hours', value: '6h' },
  { label: 'Last 24 hours', value: '1d' },
  { label: 'Last 7 days', value: '7d' },
] as const;

export function LogSearchInput({
  value,
  onChange,
  onQueryChange,
  metadataKeys,
  placeholder = 'Search log messages...',
  showAutocomplete = true,
  className,
  knownValues = {},
}: LogSearchInputProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [isOpen, setIsOpen] = useState(false);
  const [cursorPosition, setCursorPosition] = useState(0);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [showHelp, setShowHelp] = useState(false);
  const [parsedQuery, setParsedQuery] = useState<ParsedLogQuery>(() =>
    parseLogQuery(value),
  );

  // Parse the query when value changes
  useEffect(() => {
    const parsed = parseLogQuery(value);
    setParsedQuery(parsed);
    onQueryChange(parsed);
  }, [value, onQueryChange]);

  // Get autocomplete context and suggestions
  const autocompleteContext = getAutocompleteContext(value, cursorPosition);
  const suggestions = getSuggestions(
    autocompleteContext,
    metadataKeys,
    knownValues,
  );

  // Reset selected index when suggestions change
  useEffect(() => {
    setSelectedIndex(0);
  }, [suggestions.length, autocompleteContext.mode]);

  const handleInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      onChange(e.target.value);
      setCursorPosition(e.target.selectionStart || 0);
      if (showAutocomplete && e.target.value.length > 0) {
        setIsOpen(true);
      }
    },
    [onChange, showAutocomplete],
  );

  const appendFilter = useCallback(
    (filter: string) => {
      const newValue = value ? `${value.trim()} ${filter}` : filter;
      onChange(newValue);
      // Focus input and move cursor to end
      setTimeout(() => {
        inputRef.current?.focus();
        const len = newValue.length;
        inputRef.current?.setSelectionRange(len, len);
        setCursorPosition(len);
      }, 0);
    },
    [value, onChange],
  );

  const handleSuggestionSelect = useCallback(
    (suggestionValue: string) => {
      const beforeCursor = value.slice(0, cursorPosition);
      const afterCursor = value.slice(cursorPosition);

      // Find where to insert the suggestion
      const lastSpaceIndex = beforeCursor.lastIndexOf(' ');
      const lastColonIndex = beforeCursor.lastIndexOf(':');

      let newValue: string;
      let newCursorPos: number;

      if (lastColonIndex > lastSpaceIndex) {
        // Replace partial value after colon
        const beforeColon = beforeCursor.slice(0, lastColonIndex + 1);
        newValue =
          beforeColon + suggestionValue + ' ' + afterCursor.trimStart();
        newCursorPos = beforeColon.length + suggestionValue.length + 1;
      } else {
        // Replace partial key
        const beforeWord =
          lastSpaceIndex >= 0 ? beforeCursor.slice(0, lastSpaceIndex + 1) : '';
        newValue = beforeWord + suggestionValue + afterCursor.trimStart();
        newCursorPos = beforeWord.length + suggestionValue.length;
      }

      onChange(newValue);
      setCursorPosition(newCursorPos);
      setIsOpen(false);

      // Focus back on input
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

  const handleRemoveFilter = useCallback(
    (tokenToRemove: QueryToken) => {
      // Remove the token from the query string
      const newValue =
        value.slice(0, tokenToRemove.position.start) +
        value.slice(tokenToRemove.position.end);
      onChange(newValue.trim().replace(/\s+/g, ' '));
    },
    [value, onChange],
  );

  const filterTokens = parsedQuery.tokens.filter(
    (t): t is QueryToken & { type: 'filter'; key: string } =>
      t.type === 'filter' && t.key !== undefined,
  );

  // Get level values from knownValues or use defaults
  const levelValues = knownValues['level'] || [...LOG_LEVELS];

  // Get custom metadata keys (excluding reserved ones)
  const customMetadataKeys = metadataKeys.filter(
    (k) => !['level', 'after', 'before'].includes(k),
  );

  return (
    <div className={cn('space-y-2', className)}>
      {/* Search input with autocomplete */}
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
                if (suggestions.length > 0) {
                  setIsOpen(true);
                }
              }}
              onBlur={() => setTimeout(() => setIsOpen(false), 200)}
              onClick={(e) => {
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
          className="w-[var(--radix-popover-trigger-width)] p-0"
          align="start"
          onOpenAutoFocus={(e) => e.preventDefault()}
        >
          <Command>
            <CommandList>
              <CommandEmpty>No suggestions</CommandEmpty>
              <CommandGroup
                heading={
                  autocompleteContext.mode === 'key' ? 'Filter Keys' : 'Values'
                }
              >
                {suggestions.map((suggestion, index) => (
                  <CommandItem
                    key={`${suggestion.value}-${index}`}
                    onSelect={() => handleSuggestionSelect(suggestion.value)}
                    className={cn(
                      index === selectedIndex &&
                        'bg-accent text-accent-foreground',
                    )}
                    onMouseEnter={() => setSelectedIndex(index)}
                  >
                    <span className="font-mono text-sm">{suggestion.label}</span>
                    {suggestion.description && (
                      <span className="ml-2 text-xs text-muted-foreground">
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

      {/* Filter dropdowns row */}
      <div className="flex flex-wrap items-center gap-2">
        {/* Level dropdown */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="h-7 text-xs gap-1">
              <LayersIcon className="h-3 w-3" />
              Level
              <ChevronDownIcon className="h-3 w-3" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuLabel className="text-xs">Log Level</DropdownMenuLabel>
            <DropdownMenuSeparator />
            {levelValues.map((level) => (
              <DropdownMenuItem
                key={level}
                onClick={() => appendFilter(`level:${level}`)}
                className="font-mono text-xs"
              >
                <span
                  className={cn(
                    'mr-2 h-2 w-2 rounded-full',
                    level === 'error' || level === 'fatal'
                      ? 'bg-destructive'
                      : level === 'warn'
                        ? 'bg-yellow-500'
                        : level === 'info'
                          ? 'bg-blue-500'
                          : 'bg-muted-foreground',
                  )}
                />
                {level}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

        {/* Time range dropdown */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="h-7 text-xs gap-1">
              <ClockIcon className="h-3 w-3" />
              Time
              <ChevronDownIcon className="h-3 w-3" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuLabel className="text-xs">
              Time Range
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            {TIME_PRESETS.map((preset) => (
              <DropdownMenuItem
                key={preset.value}
                onClick={() => appendFilter(`after:${preset.value}`)}
                className="text-xs"
              >
                {preset.label}
              </DropdownMenuItem>
            ))}
            <DropdownMenuSeparator />
            <DropdownMenuLabel className="text-xs text-muted-foreground">
              Or use: after:2024-01-01 before:2024-12-31
            </DropdownMenuLabel>
          </DropdownMenuContent>
        </DropdownMenu>

        {/* Custom metadata dropdowns */}
        {customMetadataKeys.slice(0, 3).map((key) => {
          const values = knownValues[key] || [];
          if (values.length === 0) return null;

          return (
            <DropdownMenu key={key}>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-7 text-xs gap-1"
                >
                  {key}
                  <ChevronDownIcon className="h-3 w-3" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="max-h-64 overflow-y-auto">
                <DropdownMenuLabel className="text-xs">{key}</DropdownMenuLabel>
                <DropdownMenuSeparator />
                {values.slice(0, 20).map((val) => (
                  <DropdownMenuItem
                    key={val}
                    onClick={() => appendFilter(`${key}:${val}`)}
                    className="font-mono text-xs"
                  >
                    {val}
                  </DropdownMenuItem>
                ))}
                {values.length > 20 && (
                  <DropdownMenuLabel className="text-xs text-muted-foreground">
                    +{values.length - 20} more
                  </DropdownMenuLabel>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          );
        })}

        {/* Help button */}
        <Popover open={showHelp} onOpenChange={setShowHelp}>
          <PopoverTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-7 p-0 text-muted-foreground"
            >
              <InfoCircledIcon className="h-3.5 w-3.5" />
            </Button>
          </PopoverTrigger>
          <PopoverContent align="start" className="w-80 text-sm">
            <div className="space-y-3">
              <div>
                <h4 className="font-medium mb-1">Search Syntax</h4>
                <p className="text-xs text-muted-foreground">
                  Type to search log messages, or use filters:
                </p>
              </div>
              <div className="space-y-2 font-mono text-xs">
                <div className="flex justify-between">
                  <code className="bg-muted px-1 rounded">level:error</code>
                  <span className="text-muted-foreground">Filter by level</span>
                </div>
                <div className="flex justify-between">
                  <code className="bg-muted px-1 rounded">after:1h</code>
                  <span className="text-muted-foreground">Last hour</span>
                </div>
                <div className="flex justify-between">
                  <code className="bg-muted px-1 rounded">after:2024-01-01</code>
                  <span className="text-muted-foreground">Since date</span>
                </div>
                <div className="flex justify-between">
                  <code className="bg-muted px-1 rounded">env:production</code>
                  <span className="text-muted-foreground">Metadata filter</span>
                </div>
              </div>
              <div className="text-xs text-muted-foreground pt-2 border-t">
                <p>
                  Combine filters: <code>level:error after:1h env:prod</code>
                </p>
                <p className="mt-1">Use ↑↓ to navigate suggestions, Enter to select</p>
              </div>
            </div>
          </PopoverContent>
        </Popover>
      </div>

      {/* Show parse errors */}
      {!parsedQuery.isValid && parsedQuery.errors.length > 0 && (
        <div className="text-xs text-destructive">
          {parsedQuery.errors.join(', ')}
        </div>
      )}

      {/* Show active filters as badges */}
      {filterTokens.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {filterTokens.map((token, index) => (
            <Badge
              key={`${token.key}-${index}`}
              variant="secondary"
              className="font-mono text-xs gap-1 pr-1"
            >
              <span className="text-muted-foreground">{token.key}:</span>
              {token.value}
              <button
                type="button"
                onClick={() => handleRemoveFilter(token)}
                className="ml-1 hover:text-destructive"
              >
                <Cross2Icon className="h-3 w-3" />
              </button>
            </Badge>
          ))}
        </div>
      )}
    </div>
  );
}
