import { useMemo } from 'react';
import { LogLine } from '@/lib/api/generated/cloud/data-contracts';

/**
 * Extract unique metadata keys from log data
 */
export function useMetadataKeys(logs: LogLine[]): string[] {
  return useMemo(() => {
    const keys = new Set<string>();

    for (const log of logs) {
      if (log.metadata && typeof log.metadata === 'object') {
        for (const key of Object.keys(log.metadata)) {
          keys.add(key);
        }
      }
    }

    return Array.from(keys).sort();
  }, [logs]);
}

/**
 * Extract unique values for each metadata key (for value autocomplete)
 */
export function useMetadataValues(
  logs: LogLine[],
): Record<string, string[]> {
  return useMemo(() => {
    const values: Record<string, Set<string>> = {};

    for (const log of logs) {
      if (log.metadata && typeof log.metadata === 'object') {
        for (const [key, value] of Object.entries(
          log.metadata as Record<string, unknown>,
        )) {
          if (typeof value === 'string') {
            if (!values[key]) {
              values[key] = new Set();
            }
            values[key].add(value);
          }
        }
      }

      // Also track level values
      if (log.level) {
        if (!values['level']) {
          values['level'] = new Set();
        }
        values['level'].add(log.level);
      }
    }

    // Convert Sets to sorted arrays
    const result: Record<string, string[]> = {};
    for (const [key, valueSet] of Object.entries(values)) {
      result[key] = Array.from(valueSet).sort().slice(0, 50); // Limit values
    }

    return result;
  }, [logs]);
}
