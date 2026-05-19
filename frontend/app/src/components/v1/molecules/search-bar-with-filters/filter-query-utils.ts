import type { SearchSuggestion } from './search-bar-with-filters';

export interface FilterSuggestion extends SearchSuggestion<'key' | 'value'> {
  type: 'key' | 'value';
  label: string;
  value: string;
  description?: string;
  color?: string;
}

export type AutocompleteMode = 'key' | 'value' | 'none';

export interface AutocompleteState<
  T extends FilterSuggestion = FilterSuggestion,
> {
  mode: AutocompleteMode;
  suggestions: T[];
}

export interface FilterToken {
  key: string;
  value: string;
}

export interface TextToken {
  text: string;
}

export type QueryToken = FilterToken | TextToken;

export function isFilterToken(token: QueryToken): token is FilterToken {
  return 'key' in token;
}

const TOKEN_REGEX = /(\S+?):(\S+)|(\S+)/g;

export function tokenizeFilterQuery(query: string): QueryToken[] {
  const tokens: QueryToken[] = [];
  let match;

  while ((match = TOKEN_REGEX.exec(query)) !== null) {
    const [, key, value, text] = match;

    if (key && value !== undefined) {
      tokens.push({ key, value });
    } else if (text) {
      tokens.push({ text });
    }
  }

  return tokens;
}

export function applySuggestion(
  query: string,
  suggestion: FilterSuggestion,
  filterKeys: FilterSuggestion[],
): string {
  const trimmed = query.trimEnd();
  const words = trimmed.split(' ');
  const lastWord = words.pop() || '';

  if (suggestion.type === 'value') {
    const prefix = lastWord.slice(0, lastWord.indexOf(':') + 1);
    words.push(prefix + suggestion.value);
  } else {
    const isPartialKey = filterKeys.some((key) =>
      key.value.startsWith(lastWord.toLowerCase()),
    );
    if (lastWord && isPartialKey) {
      words.push(suggestion.value);
    } else {
      words.push(lastWord, suggestion.value);
    }
  }

  return words.filter(Boolean).join(' ');
}
