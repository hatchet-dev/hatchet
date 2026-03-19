import { SPAN_STATUSES, type ParsedTraceQuery, type SpanStatus } from './types';

export function parseTraceQuery(query: string): ParsedTraceQuery {
  const errors: string[] = [];
  const textParts: string[] = [];
  const attributes: [string, string][] = [];
  let status: SpanStatus | undefined;

  const tokenRegex = /(\S+?):(\S+)|(\S+)/g;
  let match;

  while ((match = tokenRegex.exec(query)) !== null) {
    const [, key, value, text] = match;

    if (key && value !== undefined) {
      if (key.toLowerCase() === 'status') {
        const normalized = value.toLowerCase();
        if (SPAN_STATUSES.includes(normalized as SpanStatus)) {
          status = normalized as SpanStatus;
        } else {
          errors.push(`Invalid status: "${value}"`);
        }
      } else {
        attributes.push([key, value]);
      }
    } else if (text) {
      textParts.push(text);
    }
  }

  return {
    search: textParts.length > 0 ? textParts.join(' ') : undefined,
    status,
    attributes,
    raw: query,
    isValid: errors.length === 0,
    errors,
  };
}
