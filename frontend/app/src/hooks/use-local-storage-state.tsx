import { useEffect, useState } from 'react';

const LOCAL_STORAGE_EVENT = 'hatchet:local-storage';

type SetStateValue<T> = T | ((prev: T) => T);

export function useLocalStorageState<T>(
  key: string,
  defaultValue: T,
): [T, (value: SetStateValue<T>) => void] {
  const [state, setState] = useState<T>(() => {
    try {
      const item = window.localStorage.getItem(key);
      return item ? JSON.parse(item) : defaultValue;
    } catch (error) {
      return defaultValue;
    }
  });

  const setValue = (value: SetStateValue<T>) => {
    try {
      setState((prev) => {
        const next =
          typeof value === 'function' ? (value as (prev: T) => T)(prev) : value;
        window.localStorage.setItem(key, JSON.stringify(next));
        // Ensure other hook instances in the same tab update too (the native `storage`
        // event does not fire within the same document).
        window.dispatchEvent(
          new CustomEvent(LOCAL_STORAGE_EVENT, {
            detail: { key, value: next },
          }),
        );
        return next;
      });
    } catch (error) {
      console.warn(`Error setting localStorage key "${key}":`, error);
    }
  };

  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === key && e.newValue) {
        try {
          setState(JSON.parse(e.newValue));
        } catch (error) {
          console.warn(
            `Error parsing localStorage change for key "${key}":`,
            error,
          );
        }
      }
    };

    const handleInTabChange = (e: Event) => {
      const custom = e as CustomEvent<{ key?: string; value?: T }>;
      if (custom.detail?.key !== key) {
        return;
      }

      setState(custom.detail.value as T);
    };

    window.addEventListener('storage', handleStorageChange);
    window.addEventListener(LOCAL_STORAGE_EVENT, handleInTabChange);
    return () => {
      window.removeEventListener('storage', handleStorageChange);
      window.removeEventListener(LOCAL_STORAGE_EVENT, handleInTabChange);
    };
  }, [key]);

  return [state, setValue];
}
