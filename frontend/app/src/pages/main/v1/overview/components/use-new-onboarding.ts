import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { appRoutes } from '@/router';
import { useNavigate } from '@tanstack/react-router';
import { useCallback, useEffect } from 'react';

// Temporary local-storage flag gating the entire new onboarding surface.
// Default OFF, so real users see no change while this is built out.
export const newOnboardingStorageKey = 'hatchet:new-onboarding';

// TODO: Replace this temporary local-storage flag with
//   useIsFeatureEnabled(FeatureFlagId.NewOnboardingEnabled, false)
// once that flag is added to the backend feature-flag enum and the client
// is regenerated.
export function useNewOnboardingEnabled(): boolean {
  const [enabled, setEnabled] = useLocalStorageState<boolean>(
    newOnboardingStorageKey,
    false,
  );

  // Allow previewing the feature via the URL: ?newOnboarding=1 turns it on,
  // ?newOnboarding=0 turns it off. Runs once on mount.
  useEffect(() => {
    let override: string | null = null;
    try {
      override = new URLSearchParams(window.location.search).get(
        'newOnboarding',
      );
    } catch {
      override = null;
    }

    if (override === '1') {
      setEnabled(true);
    } else if (override === '0') {
      setEnabled(false);
    }
    // Only intended to run on mount; setEnabled is stable enough for this.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // !!! TEMPORARY: forced ON for everyone for local end-to-end testing. !!!
  // Revert this to `return enabled;` (and drop the `void enabled` line)
  // before merge. The localStorage + ?newOnboarding=1 preview logic above is
  // preserved so the revert is one line.
  void enabled;
  return true;
}

// Opens the "run your first task" onboarding for a tenant. The onboarding is
// URL-controlled (the tenant onboarding route), so opening it is a navigation:
// that is what makes it refresh-safe, linkable and tenant-scoped.
export function useOpenOnboarding(tenantId: string | undefined) {
  const navigate = useNavigate();

  return useCallback(() => {
    if (!tenantId) {
      return;
    }
    navigate({
      to: appRoutes.tenantOnboardingRoute.to,
      params: { tenant: tenantId },
    });
  }, [navigate, tenantId]);
}
