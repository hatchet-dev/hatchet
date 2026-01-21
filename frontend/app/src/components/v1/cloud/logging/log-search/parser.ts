import { LOG_LEVELS, LogLevel, ParsedLogQuery } from './types';

export function parseLogQuery(query: string): ParsedLogQuery {
  const errors: string[] = [];
  const textParts: string[] = [];
  let level: LogLevel | undefined;

  const tokenRegex = /(\S+?):(\S+)|(\S+)/g;
  let match;

  while ((match = tokenRegex.exec(query)) !== null) {
    const [, key, value, text] = match;

    if (key && value !== undefined) {
      if (key.toLowerCase() === 'level') {
        const normalizedLevel = value.toLowerCase();
        if (LOG_LEVELS.includes(normalizedLevel as LogLevel)) {
          level = normalizedLevel as LogLevel;
        } else {
          errors.push(`Invalid log level: "${value}"`);
        }
      } else {
        textParts.push(`${key}:${value}`);
      }
    } else if (text) {
      textParts.push(text);
    }
  }

  return {
    search: textParts.length > 0 ? textParts.join(' ') : undefined,
    level,
    raw: query,
    isValid: errors.length === 0,
    errors,
  };
}
