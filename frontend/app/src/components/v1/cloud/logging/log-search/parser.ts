import { LOG_LEVELS, LogLevel, ParsedLogQuery } from './types';

export function parseLogQuery(query: string): ParsedLogQuery {
  const errors: string[] = [];
  const textParts: string[] = [];
  let level: LogLevel | undefined;
  let attempt: number | undefined;

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
      } else if (key.toLowerCase() === 'attempt') {
        const parsedAttempt = parseInt(value, 10);
        if (!isNaN(parsedAttempt) && parsedAttempt > 0) {
          attempt = parsedAttempt;
        } else {
          errors.push(`Invalid attempt number: "${value}"`);
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
    attempt,
    raw: query,
    isValid: errors.length === 0,
    errors,
  };
}
