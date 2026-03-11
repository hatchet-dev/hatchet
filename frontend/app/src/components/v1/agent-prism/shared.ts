import { useEffect, useState } from 'react';

// TYPES

export type ComponentSize =
  | '4'
  | '5'
  | '6'
  | '7'
  | '8'
  | '9'
  | '10'
  | '11'
  | '12'
  | '16';

// CONSTANTS

export const ROUNDED_CLASSES = {
  none: 'rounded-none',
  sm: 'rounded-sm',
  md: 'rounded-md',
  lg: 'rounded-lg',
  full: 'rounded-full',
};

// UTILS

export const useIsMobile = () => {
  const isMounted = useIsMounted();

  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    // TODO: replace with something more beautiful and correct (tailwind screens?)
    const mediaQuery = window.matchMedia('(max-width: 1023px)');

    const handleChange = (e: MediaQueryListEvent | MediaQueryList) => {
      setIsMobile(e.matches);
    };

    handleChange(mediaQuery);

    mediaQuery.addEventListener('change', handleChange);

    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  return isMounted ? isMobile : false;
};

export const useIsMounted = () => {
  const [isMounted, setIsMounted] = useState(false);

  useEffect(() => {
    setIsMounted(true);
  }, []);

  return isMounted;
};
