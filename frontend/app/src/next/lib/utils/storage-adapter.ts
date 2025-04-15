import * as React from 'react';
import { useSearchParams as useRouterSearchParams } from 'react-router-dom';

// Generic interface for storage adapters
export interface StorageAdapter<T = any> {
  getValue<K extends keyof T>(key: string, defaultValue: T[K]): T[K];
  setValue<K extends keyof T>(key: string, value: T[K]): void;
  getValues(): T;
}

// React state-based storage adapter
class StateAdapter<T extends Record<string, any>> implements StorageAdapter<T> {
  private setState: React.Dispatch<React.SetStateAction<T>>;

  private state: T;

  constructor(
    initialState: T,
    setState: React.Dispatch<React.SetStateAction<T>>,
  ) {
    this.state = initialState;
    this.setState = setState;
  }

  getValue<K extends keyof T>(key: string, defaultValue: T[K]): T[K] {
    return (this.state[key as keyof T] as T[K]) ?? defaultValue;
  }

  setValue<K extends keyof T>(key: string, value: T[K]): void {
    this.setState(
      (prev) =>
        ({
          ...prev,
          [key]: value,
        }) as T,
    );
  }

  getValues(): T {
    return this.state;
  }
}

// URL query params-based storage adapter
class QueryParamsStorageAdapter<T extends Record<string, any>>
  implements StorageAdapter<T>
{
  private searchParams: URLSearchParams;

  private setSearchParams: (
    nextInit: URLSearchParams | ((prev: URLSearchParams) => URLSearchParams),
  ) => void;

  private prefix: string;

  constructor(
    searchParams: URLSearchParams,
    setSearchParams: (
      nextInit: URLSearchParams | ((prev: URLSearchParams) => URLSearchParams),
    ) => void,
    prefix = '',
  ) {
    this.searchParams = searchParams;
    this.setSearchParams = setSearchParams;
    this.prefix = prefix;
  }

  getValue<K extends keyof T>(key: string, defaultValue: T[K]): T[K] {
    const paramKey = this.prefix ? `${this.prefix}${key}` : key;
    const value = this.searchParams.get(paramKey);

    if (value === null) {
      return defaultValue;
    }

    // Handle different types
    if (typeof defaultValue === 'number') {
      return Number(value) as unknown as T[K];
    } else if (typeof defaultValue === 'boolean') {
      return (value === 'true') as unknown as T[K];
    } else {
      try {
        // Try to parse as JSON for objects and arrays
        return JSON.parse(value) as T[K];
      } catch (e) {
        // If not JSON, return as string
        return value as unknown as T[K];
      }
    }
  }

  setValue<K extends keyof T>(key: string, value: T[K]): void {
    const paramKey = this.prefix ? `${this.prefix}${key}` : key;

    this.setSearchParams((prev) => {
      const newParams = new URLSearchParams(prev);

      if (value === undefined || value === null) {
        newParams.delete(paramKey);
      } else if (typeof value === 'object') {
        // Convert objects to JSON strings
        newParams.set(paramKey, JSON.stringify(value));
      } else {
        // For primitives, convert to string
        newParams.set(paramKey, String(value));
      }

      return newParams;
    });
  }

  getValues(): T {
    const result = {} as T;
    const prefixLength = this.prefix.length;

    for (const [key, value] of this.searchParams.entries()) {
      if (!this.prefix || key.startsWith(this.prefix)) {
        const resultKey = this.prefix ? key.substring(prefixLength) : key;

        try {
          // Try to parse JSON values
          result[resultKey as keyof T] = JSON.parse(value);
        } catch (e) {
          // If not JSON, use the raw value
          result[resultKey as keyof T] = value as any;
        }
      }
    }

    return result;
  }
}

// Type for the storage hook options
export interface UseStateAdapterOptions {
  type?: 'state' | 'query';
  prefix?: string;
}

/**
 * A hook that provides storage functionality using either React state or URL query parameters
 * @param initialValues The initial values for the storage
 * @param options Configuration options for the storage
 * @returns A storage adapter instance
 */
export function useStateAdapter<T extends Record<string, any>>(
  initialValues: T,
  options: UseStateAdapterOptions = {},
): StorageAdapter<T> {
  const { type = 'state', prefix = '' } = options;

  // State for the state-based adapter
  const [stateValues, setStateValues] = React.useState<T>(initialValues);

  // Query params for the query-based adapter
  const [searchParams, setSearchParams] = useRouterSearchParams();

  // Create and memoize the appropriate adapter
  return React.useMemo(() => {
    if (type === 'query') {
      return new QueryParamsStorageAdapter<T>(
        searchParams,
        setSearchParams,
        prefix,
      );
    }
    return new StateAdapter<T>(stateValues, setStateValues);
  }, [
    type,
    stateValues,
    setStateValues,
    searchParams,
    setSearchParams,
    prefix,
  ]);
}
