import { SPAN_STATUSES, type ParsedTraceQuery, type SpanStatus } from './types';
import {
  tokenizeFilterQuery,
  isFilterToken,
} from '@/components/v1/molecules/search-bar-with-filters/filter-query-utils';

export function parseTraceQuery(query: string): ParsedTraceQuery {
  const errors: string[] = [];
  const textParts: string[] = [];
  const attributes: [string, string][] = [];
  let status: SpanStatus | undefined;

  for (const token of tokenizeFilterQuery(query)) {
    if (isFilterToken(token)) {
      const { key, value } = token;
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
    } else {
      textParts.push(token.text);
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
