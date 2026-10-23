import { appRoutes } from '@/router';
import { useNavigate } from '@tanstack/react-router';
import { useCallback } from 'react';

// The onboarding is URL-controlled (the tenant onboarding route), so opening it
// is a navigation: that is what makes it refresh-safe and tenant-scoped.
export function useOpenOnboarding(tenantId: string) {
  const navigate = useNavigate();

  return useCallback(
    () =>
      navigate({
        to: appRoutes.tenantOnboardingRoute.to,
        params: { tenant: tenantId },
      }),
    [navigate, tenantId],
  );
}
