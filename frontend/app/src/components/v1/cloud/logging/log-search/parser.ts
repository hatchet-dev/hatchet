import { parse, parseISO, isValid as isValidDate, formatISO } from 'date-fns';
import { ParsedLogQuery, QueryToken } from './types';

/**
 * Parse date strings with various formats
 * Supports: ISO 8601, YYYY-MM-DD, relative dates (1h, 2d, 1w)
 */
export function parseDateValue(value: string): string | null {
  // Try ISO 8601 first
  const isoDate = parseISO(value);
  if (isValidDate(isoDate)) {
    return formatISO(isoDate);
  }

  // Try common date formats
  const formats = [
    'yyyy-MM-dd',
    'yyyy-MM-dd HH:mm',
    'yyyy-MM-dd HH:mm:ss',
    'MM/dd/yyyy',
    'dd/MM/yyyy',
  ];

  for (const fmt of formats) {
    const parsed = parse(value, fmt, new Date());
    if (isValidDate(parsed)) {
      return formatISO(parsed);
    }
  }

  // Handle relative dates like "1h", "2d", "1w"
  const relativeMatch = value.match(/^(\d+)([smhdw])$/);
  if (relativeMatch) {
    const [, amount, unit] = relativeMatch;
    const now = new Date();
    const ms: Record<string, number> = {
      s: 1000,
      m: 60 * 1000,
      h: 60 * 60 * 1000,
      d: 24 * 60 * 60 * 1000,
      w: 7 * 24 * 60 * 60 * 1000,
    };

    const date = new Date(now.getTime() - parseInt(amount) * (ms[unit] || 0));
    return formatISO(date);
  }

  return null;
}

/**
 * Tokenize the query string into individual tokens
 */
export function tokenize(query: string): QueryToken[] {
  const tokens: QueryToken[] = [];

  // Regex to match key:value pairs (with optional quoted values) or plain text
  const tokenRegex = /(\S+?):((?:"[^"]*")|(?:\S+))|(\S+)/g;
  let match;

  while ((match = tokenRegex.exec(query)) !== null) {
    const [fullMatch, key, value, text] = match;
    const start = match.index;
    const end = start + fullMatch.length;

    if (key && value !== undefined) {
      // Filter token (key:value)
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
      // Text token
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

/**
 * Parse a query string into a structured ParsedLogQuery
 */
export function parseLogQuery(query: string): ParsedLogQuery {
  const tokens = tokenize(query);
  const errors: string[] = [];

  const result: ParsedLogQuery = {
    metadata: {},
    tokens,
    raw: query,
    isValid: true,
    errors: [],
  };

  const textParts: string[] = [];

  for (const token of tokens) {
    if (token.type === 'text') {
      textParts.push(token.value);
      continue;
    }

    const key = token.key!;
    const value = token.value;

    if (key === 'after' || key === 'before') {
      const parsedDate = parseDateValue(value);
      if (parsedDate) {
        result[key] = parsedDate;
      } else {
        errors.push(`Invalid date format for ${key}: "${value}"`);
      }
    } else if (key === 'level') {
      result.level = value.toLowerCase();
    } else {
      // Metadata filter
      result.metadata[key] = value;
    }
  }

  // Combine text parts for free text search
  if (textParts.length > 0) {
    result.search = textParts.join(' ');
  }

  result.isValid = errors.length === 0;
  result.errors = errors;

  return result;
}

/**
 * Convert ParsedLogQuery back to query string
 */
export function serializeLogQuery(query: Partial<ParsedLogQuery>): string {
  const parts: string[] = [];

  if (query.search) {
    parts.push(query.search);
  }

  if (query.after) {
    parts.push(`after:${query.after}`);
  }

  if (query.before) {
    parts.push(`before:${query.before}`);
  }

  if (query.level) {
    parts.push(`level:${query.level}`);
  }

  if (query.metadata) {
    for (const [key, value] of Object.entries(query.metadata)) {
      const needsQuotes = value.includes(' ');
      parts.push(`${key}:${needsQuotes ? `"${value}"` : value}`);
    }
  }

  return parts.join(' ');
}

/**
 * Get the token at a specific cursor position
 */
export function getTokenAtPosition(
  tokens: QueryToken[],
  position: number,
): QueryToken | null {
  for (const token of tokens) {
    if (position >= token.position.start && position <= token.position.end) {
      return token;
    }
  }
  return null;
}
