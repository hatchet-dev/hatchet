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

export interface SearchSuggestion<TType extends string = string> {
  type: TType;
  label: string;
  value: string;
  description?: string;
  color?: string;
  metadata?: Record<string, unknown>;
}

export interface AutocompleteResult<TSuggestion extends SearchSuggestion> {
  suggestions: TSuggestion[];
  mode?: string;
}

export interface FilterChip {
  key: string;
  label: string;
  description?: string;
  // For complex filters that need custom input components
  customInput?: React.ComponentType<CustomFilterInputProps>;
  // For filters that need multi-step input (e.g., time ranges)
  requiresCustomInput?: boolean;
}

export interface CustomFilterInputProps {
  filterKey: string;
  currentValue: string;
  onComplete: (value: string) => void;
  onCancel: () => void;
}

export interface SearchBarWithFiltersProps<
  TSuggestion extends SearchSuggestion,
> {
  // Value control
  value: string;
  onChange: (value: string) => void;
  onSubmit?: (value: string) => void;

  // Autocomplete functions (domain-specific)
  getAutocomplete: (
    query: string,
    context?: unknown,
  ) => AutocompleteResult<TSuggestion>;
  applySuggestion: (query: string, suggestion: TSuggestion) => string;
  autocompleteContext?: unknown;

  // Custom rendering
  renderSuggestion?: (
    suggestion: TSuggestion,
    isSelected: boolean,
  ) => React.ReactNode;

  // Filter chips
  filterChips?: FilterChip[];
  onFilterChipClick?: (filterKey: string) => void;

  // UI customization
  placeholder?: string;
  className?: string;
  searchIcon?: React.ReactNode;
  clearIcon?: React.ReactNode;
  popoverClassName?: string;
}

// ============================================================================
// Component
// ============================================================================

export function SearchBarWithFilters<TSuggestion extends SearchSuggestion>({
  value,
  onChange,
  onSubmit,
  getAutocomplete,
  applySuggestion,
  autocompleteContext,
  renderSuggestion,
  filterChips,
  onFilterChipClick,
  placeholder = 'Search...',
  className,
  searchIcon,
  clearIcon,
  popoverClassName,
}: SearchBarWithFiltersProps<TSuggestion>) {
  const inputRef = useRef<HTMLInputElement>(null);
  const blurTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const justSelectedRef = useRef(false);
  const [isOpen, setIsOpen] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState<number>();
  const [localValue, setLocalValue] = useState(value);
  const prevSuggestionsRef = useRef<TSuggestion[]>([]);
  const prevLocalValueRef = useRef(value);

  useEffect(() => {
    setLocalValue(value);
  }, [value]);

  const { suggestions } = getAutocomplete(localValue, autocompleteContext);

  // Auto-select first suggestion when suggestions change (e.g., from keys to values)
  useEffect(() => {
    const suggestionsChanged =
      prevSuggestionsRef.current.length !== suggestions.length ||
      prevSuggestionsRef.current.some(
        (prev, i) => prev.value !== suggestions[i]?.value,
      );

    if (suggestionsChanged) {
      // Auto-select first suggestion if any exist, otherwise clear selection
      setSelectedIndex(suggestions.length > 0 ? 0 : undefined);

      // Don't automatically open dropdown - let explicit user actions control it
      // (typing, focus, space key, etc.)

      prevSuggestionsRef.current = suggestions;
    }
  }, [suggestions]);

  // Open dropdown when space is added (for adding new filters)
  useEffect(() => {
    const justAddedSpace =
      localValue.endsWith(' ') &&
      !prevLocalValueRef.current.endsWith(' ') &&
      localValue.length > prevLocalValueRef.current.length;

    if (justAddedSpace && suggestions.length > 0) {
      setIsOpen(true);
    }

    prevLocalValueRef.current = localValue;
  }, [localValue, suggestions.length]);

  const submitSearch = useCallback(() => {
    if (onSubmit) {
      onSubmit(localValue);
    } else {
      onChange(localValue);
    }
  }, [localValue, onChange, onSubmit]);

  const handleFilterChipClick = useCallback(
    (filterKey: string) => {
      if (onFilterChipClick) {
        onFilterChipClick(filterKey);
        return;
      }

      const newValue = localValue ? `${localValue} ${filterKey}` : filterKey;
      setLocalValue(newValue);
      // Don't call onChange - user is still building the filter
      setIsOpen(true);
      setTimeout(() => {
        const input = inputRef.current;
        if (input) {
          input.focus();
          input.setSelectionRange(newValue.length, newValue.length);
        }
      }, 0);
    },
    [localValue, onFilterChipClick],
  );

  const handleSelect = useCallback(
    (index: number) => {
      const suggestion = suggestions[index];
      if (suggestion) {
        const newValue = applySuggestion(localValue, suggestion);
        setLocalValue(newValue);

        // Only update parent state for 'value' type suggestions (complete filters)
        // This triggers the actual search
        if (suggestion.type === 'value') {
          onChange(newValue);
          setIsOpen(false);
          // Mark that we just selected a value to prevent reopening on refocus
          justSelectedRef.current = true;
        }
        // For 'key' type suggestions, don't update parent - user is still building the filter
        // Keep dropdown open for value selection

        setTimeout(() => {
          const input = inputRef.current;
          if (input) {
            input.focus();
            input.setSelectionRange(newValue.length, newValue.length);
          }
          // Reset the flag after focus is restored
          setTimeout(() => {
            justSelectedRef.current = false;
          }, 50);
        }, 0);
      }
    },
    [localValue, suggestions, applySuggestion, onChange],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        // If input is empty or just whitespace, always submit (don't apply suggestions)
        const isEmpty = localValue.trim() === '';
        if (isEmpty) {
          submitSearch();
          setIsOpen(false);
        } else if (
          isOpen &&
          suggestions.length > 0 &&
          selectedIndex !== undefined
        ) {
          handleSelect(selectedIndex);
        } else {
          submitSearch();
          setIsOpen(false);
        }
        return;
      }

      // Don't handle space specially - let onChange detect it and handle opening
      // This ensures suggestions have updated before opening dropdown

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
    [
      isOpen,
      suggestions.length,
      selectedIndex,
      handleSelect,
      submitSearch,
      localValue,
    ],
  );

  const defaultRenderSuggestion = useCallback(
    (suggestion: TSuggestion, isSelected: boolean) => (
      <div
        className={cn(
          'flex items-center justify-between w-full',
          isSelected && 'bg-accent text-accent-foreground',
        )}
      >
        <div className="flex items-center gap-2">
          {suggestion.color && (
            <div
              className={cn(suggestion.color, 'h-[6px] w-[6px] rounded-full')}
            />
          )}
          {suggestion.type === 'key' ? (
            <code className="px-1.5 py-0.5 bg-muted rounded text-xs font-mono">
              {suggestion.label}:
            </code>
          ) : (
            <span className="font-mono text-sm">{suggestion.label}</span>
          )}
        </div>
        {suggestion.description && (
          <span className="text-xs text-muted-foreground truncate ml-2">
            {suggestion.description}
          </span>
        )}
      </div>
    ),
    [],
  );

  const renderSuggestionContent = renderSuggestion || defaultRenderSuggestion;

  const handleClear = useCallback(() => {
    setLocalValue('');
    // Clear the search by notifying parent with empty string
    onChange('');
    inputRef.current?.focus();
    setIsOpen(false);
  }, [onChange]);

  return (
    <div className={cn('space-y-2', className)}>
      <Popover open={isOpen && suggestions.length > 0} modal={false}>
        <PopoverTrigger asChild>
          <div className="relative">
            {searchIcon || (
              <MagnifyingGlassIcon className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            )}
            <Input
              ref={inputRef}
              type="text"
              value={localValue}
              onChange={(e) => {
                // Only update local state for autocomplete
                // Don't trigger parent onChange until Enter is pressed or value suggestion is selected
                const newValue = e.target.value;
                setLocalValue(newValue);
                // Open dropdown when user types (if there are suggestions)
                if (newValue.length > 0) {
                  setIsOpen(true);
                }
              }}
              onKeyDown={handleKeyDown}
              onFocus={() => {
                if (blurTimeoutRef.current) {
                  clearTimeout(blurTimeoutRef.current);
                  blurTimeoutRef.current = null;
                }
                // Open dropdown on focus to show available filters
                // But not if we just selected a value (prevents reopening after click)
                if (!justSelectedRef.current) {
                  setIsOpen(true);
                }
              }}
              onBlur={() => {
                blurTimeoutRef.current = setTimeout(() => {
                  setIsOpen(false);
                  blurTimeoutRef.current = null;
                }, 200);
              }}
              placeholder={placeholder}
              className="pl-9 pr-8"
              data-cy="search-bar-input"
            />
            {localValue && (
              <button
                type="button"
                onClick={handleClear}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                data-cy="search-bar-clear"
              >
                {clearIcon || <Cross2Icon className="h-4 w-4" />}
              </button>
            )}
          </div>
        </PopoverTrigger>
        <PopoverContent
          className={cn(
            'w-[var(--radix-popover-trigger-width)] min-w-[320px] p-0',
            popoverClassName,
          )}
          align="start"
          onOpenAutoFocus={(e) => e.preventDefault()}
          onCloseAutoFocus={(e) => e.preventDefault()}
        >
          {suggestions.length > 0 && (
            <Command
              key={suggestions.map((s) => s.value).join(',')}
              value={
                selectedIndex !== undefined
                  ? suggestions[selectedIndex]?.value
                  : undefined
              }
            >
              <CommandList
                className="max-h-[300px]"
                data-cy="search-bar-suggestions"
              >
                <CommandGroup>
                  {suggestions.map((suggestion, index) => (
                    <CommandItem
                      key={`${suggestion.value}-${index}`}
                      value={suggestion.value}
                      onSelect={() => handleSelect(index)}
                      className={cn(
                        'flex items-center justify-between',
                        selectedIndex !== undefined &&
                          index === selectedIndex &&
                          'bg-accent text-accent-foreground',
                      )}
                      onMouseEnter={() => setSelectedIndex(index)}
                      data-cy={`search-bar-suggestion-${index}`}
                    >
                      {renderSuggestionContent(
                        suggestion,
                        selectedIndex === index,
                      )}
                    </CommandItem>
                  ))}
                </CommandGroup>
              </CommandList>
            </Command>
          )}
          {filterChips && filterChips.length > 0 && (
            <div
              className={cn(
                'flex flex-col gap-2 px-3 py-2 text-xs',
                suggestions.length > 0 && 'border-t',
              )}
              data-cy="search-bar-filter-chips"
            >
              <div className="flex items-center gap-2">
                <span className="text-muted-foreground">
                  Available filters:
                </span>
                {filterChips.map((chip) => (
                  <Button
                    key={chip.key}
                    variant="outline"
                    size="xs"
                    className="h-auto px-2 py-0.5 text-xs"
                    onClick={() => handleFilterChipClick(chip.key)}
                    data-cy={`filter-chip-${chip.key.replace(':', '')}`}
                  >
                    {chip.label}
                  </Button>
                ))}
              </div>
              <div className="text-muted-foreground">
                Or type any text to search without filters
              </div>
            </div>
          )}
        </PopoverContent>
      </Popover>
    </div>
  );
}
