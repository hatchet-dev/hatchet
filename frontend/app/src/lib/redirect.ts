import { NavigateOptions, useNavigate } from '@tanstack/react-router';
import { useCallback } from 'react';

export const REDIRECT_TARGET_KEY = 'hatchet:redirect_target';

export function useRedirectOrNavigate() {
  const navigate = useNavigate();
  return useCallback(
    (fallbackOpts: NavigateOptions) => {
      const redirectTo = sessionStorage.getItem(REDIRECT_TARGET_KEY);
      if (redirectTo) {
        sessionStorage.removeItem(REDIRECT_TARGET_KEY);
        navigate({ to: redirectTo, replace: true } as never);
      } else {
        navigate(fallbackOpts as never);
      }
    },
    [navigate],
  );
}
