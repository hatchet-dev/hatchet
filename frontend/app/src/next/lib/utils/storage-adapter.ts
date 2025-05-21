import * as React from 'react';

export interface StorageAdapter<T = any> {
  getValue<K extends keyof T>(key: string, defaultValue: T[K]): T[K];
  setValue<K extends keyof T>(key: string, value: T[K]): void;
  setValues(values: Partial<T>): void;
  getValues(): T;
}

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

  setValues(values: Partial<T>): void {
    this.setState((prev) => ({ ...prev, ...values }) as T);
  }

  getValues(): T {
    return this.state;
  }
}


/**
 * A hook that provides storage functionality using either React state or URL query parameters
 * @param initialValues The initial values for the storage
 * @returns A storage adapter instance
 */
export function useStateAdapter<T extends Record<string, any>>(
  initialValues: T,
): StorageAdapter<T> {
  const [stateValues, setStateValues] = React.useState<T>(initialValues);

  return React.useMemo(() => {
    return new StateAdapter<T>(stateValues, setStateValues);
  }, [
    stateValues,
    setStateValues,
  ]);
}
