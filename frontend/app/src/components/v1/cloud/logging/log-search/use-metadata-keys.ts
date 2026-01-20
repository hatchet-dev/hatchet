import { useMemo } from 'react';
import { LogLine } from '@/lib/api/generated/cloud/data-contracts';

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

      if (log.level) {
        if (!values['level']) {
          values['level'] = new Set();
        }
        values['level'].add(log.level);
      }
    }

    const result: Record<string, string[]> = {};
    for (const [key, valueSet] of Object.entries(values)) {
      result[key] = Array.from(valueSet).sort().slice(0, 50);
    }

    return result;
  }, [logs]);
}
