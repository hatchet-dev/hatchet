import { LOG_LEVELS, LogLevel, ParsedLogQuery, QueryToken } from './types';

export function tokenize(query: string): QueryToken[] {
  const tokens: QueryToken[] = [];
  const tokenRegex = /(\S+?):((?:"[^"]*")|(?:\S+))|(\S+)/g;
  let match;

  while ((match = tokenRegex.exec(query)) !== null) {
    const [fullMatch, key, value, text] = match;
    const start = match.index;
    const end = start + fullMatch.length;

    if (key && value !== undefined) {
      const cleanValue =
        value.startsWith('"') && value.endsWith('"')
          ? value.slice(1, -1)
          : value;

      tokens.push({
        type: 'filter',
        key: key.toLowerCase(),
        value: cleanValue,
        raw: fullMatch,
        position: { start, end },
      });
    } else if (text) {
      tokens.push({
        type: 'text',
        value: text,
        raw: text,
        position: { start, end },
      });
    }
  }

  return tokens;
}

export function parseLogQuery(query: string): ParsedLogQuery {
  const tokens = tokenize(query);
  const errors: string[] = [];
  const textParts: string[] = [];

  const result: ParsedLogQuery = {
    tokens,
    raw: query,
    isValid: true,
    errors: [],
  };

  for (const token of tokens) {
    if (token.type === 'text') {
      textParts.push(token.value);
      continue;
    }

    if (token.key === 'level') {
      const level = token.value.toLowerCase();
      if (LOG_LEVELS.includes(level as LogLevel)) {
        result.level = level as LogLevel;
      } else {
        errors.push(`Invalid log level: "${token.value}"`);
      }
    } else {
      // Unknown filter key - treat as text search
      textParts.push(token.raw);
    }
  }

  if (textParts.length > 0) {
    result.search = textParts.join(' ');
  }

  result.isValid = errors.length === 0;
  result.errors = errors;

  return result;
}

export function serializeLogQuery(query: Partial<ParsedLogQuery>): string {
  const parts: string[] = [];

  if (query.search) {
    parts.push(query.search);
  }

  if (query.level) {
    parts.push(`level:${query.level}`);
  }

  return parts.join(' ');
}
