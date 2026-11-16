import { config } from '@/config';
import { NavigateOptions, useNavigate } from '@tanstack/react-router';
import { useCallback } from 'react';

export const REDIRECT_TARGET_KEY = 'hatchet:redirect_target';

// Full load, not a client navigation. Logout cancels in-flight queries, and
// the login route's session check was dying with that cancellation.
export function reloadToLogin() {
  const base = config.BASE_PATH.replace(/\/$/, '');
  window.location.assign(`${base}/auth/login`);
}

export function useRedirectOrNavigate() {
  const navigate = useNavigate();
  return useCallback(
    (fallbackOpts: NavigateOptions) => {
      const redirectTo = sessionStorage.getItem(REDIRECT_TARGET_KEY);
      if (redirectTo) {
        sessionStorage.removeItem(REDIRECT_TARGET_KEY);
        return navigate({ to: redirectTo, replace: true } as never);
      }
      return navigate(fallbackOpts as never);
    },
    [navigate],
  );
}
