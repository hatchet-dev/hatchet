import { workflowLanguageOptions } from './onboarding-options';
import {
  applyLanguageChange,
  applyTabChange,
  applyUseCaseChange,
  normalizeOnboardingState,
  onboardingStorageKey,
} from './onboarding-state';
import { OnboardingSteps } from './onboarding-steps';
import { type AvailableUseCaseKey } from './use-case-options';
import { useOnboardingProgress } from './use-onboarding-progress';
import { usePreferredSdk, type Sdk } from './use-preferred-sdk';
import { Button } from '@/components/v1/ui/button';
import { useAnalytics } from '@/hooks/use-analytics';
import useAuthDisabled from '@/hooks/use-auth-disabled';
import useCanWrite from '@/hooks/use-can-write';
import { useCurrentUser } from '@/hooks/use-current-user';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { useTenantDetails } from '@/hooks/use-tenant';
import api, { CreateAPITokenRequest } from '@/lib/api';
import { globalEmitter } from '@/lib/global-emitter';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { useMutation } from '@tanstack/react-query';
import { useEffect, useMemo, useRef, useState } from 'react';

// Maps the global SDK preference to the language the persisted onboarding
// state and command builders understand. Ruby has no command-builder language,
// so it leaves the persisted language untouched (the manual path is
// unavailable for Ruby anyway).
function sdkToWorkflowLanguage(sdk: Sdk) {
  switch (sdk) {
    case 'python':
      return workflowLanguageOptions.python.value;
    case 'typescript':
      return workflowLanguageOptions.typescript.value;
    case 'go':
      return workflowLanguageOptions.go.value;
    case 'ruby':
      return null;
  }
}

// A 100-year expiry, matching the profile-token flow on the Overview page.
const PROFILE_TOKEN_EXPIRES_IN = `${100 * 365 * 24 * 60 * 60}s`;

// Emit from anywhere (e.g. the Overview banner/footer) to open the modal.
export function openOnboarding() {
  globalEmitter.emit('open-onboarding', {});
}

// Full-screen onboarding overlay. It renders into AppLayout's content-area
// overlay slot (absolute inset-0), so it covers the page and the sidebar but
// NOT the header or banner: the nav bar and its tenant switcher stay visible
// and interactive regardless of banner height. Its z-[110] sits above the
// sidebar (z-[100], which stays mounted on desktop) yet below Radix dialogs
// (z-[200]) so the token-success dialog this modal spawns still layers on top.
//
// The body is the redesigned OnboardingSteps stepper: shared steps, a path
// fork (agent vs manual), and a converged finish. The modal owns the
// persisted onboarding state, the profile-token mutation, live progress, and
// analytics; the stepper owns navigation and the local path/agent/freeform
// selections. The inline LearnWorkflowSection flow on the Overview page is
// unchanged and still used when the onboarding flag is off.
export function OnboardingModal({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const { tenant, tenantId } = useTenantDetails();
  const { currentUser } = useCurrentUser();
  const authDisabled = useAuthDisabled();
  const { meta } = useApiMeta();
  const authDisabledToken =
    meta && 'authDisabledToken' in meta ? meta.authDisabledToken : undefined;
  const { capture } = useAnalytics();
  const canWrite = useCanWrite();

  const [sdk, setSdk] = usePreferredSdk();

  const [profileToken, setProfileToken] = useState<string | undefined>();
  const [profileTokenError, setProfileTokenError] = useState<
    string | undefined
  >();

  const containerRef = useRef<HTMLDivElement>(null);
  const hasCapturedOpen = useRef(false);

  // Shared with the inline flow through the same tenant-scoped storage key,
  // so progress is consistent whichever surface the user uses.
  const [storedOnboarding, setStoredOnboarding] = useLocalStorageState<unknown>(
    onboardingStorageKey(tenantId ?? 'unknown'),
    null,
  );
  const onboarding = useMemo(
    () => normalizeOnboardingState(storedOnboarding),
    [storedOnboarding],
  );

  const useCase: AvailableUseCaseKey = onboarding.useCase;

  // Advancing past step 1 confirms the selection so progress polling can
  // begin. Reuses applyTabChange semantics (any move off Choose use case sets
  // selectionConfirmedAt exactly once).
  const confirmSelection = () =>
    setStoredOnboarding((prev: unknown) =>
      applyTabChange(
        normalizeOnboardingState(prev),
        'install',
        new Date().toISOString(),
      ),
    );

  const progress = useOnboardingProgress(
    tenantId,
    onboarding.selectionConfirmedAt ?? undefined,
  );

  const defaultTokenName = useMemo(() => {
    const name = currentUser?.name?.trim();
    return name ? `${name}'s token` : '';
  }, [currentUser?.name]);

  const createProfileTokenMutation = useMutation({
    mutationKey: ['api-token:create:profile', tenantId],
    mutationFn: async (data: CreateAPITokenRequest) => {
      const res = await api.apiTokenCreate(tenantId!, data);
      return res.data;
    },
    onSuccess: (data) => {
      setProfileToken(data.token);
      setProfileTokenError(undefined);
      capture('onboarding_token_generated', {
        tenant_id: tenantId,
        user_email: currentUser?.email,
        token_name: `${defaultTokenName || 'Local'} (CLI profile)`,
        expires_in: PROFILE_TOKEN_EXPIRES_IN,
        source: 'onboarding_modal_profile_step',
      });
    },
    onError: () => {
      setProfileTokenError('Failed to generate token. Please try again.');
    },
  });

  const handleGenerateProfileToken = () => {
    setProfileTokenError(undefined);
    createProfileTokenMutation.mutate({
      name: defaultTokenName ? `${defaultTokenName} (CLI)` : 'CLI token',
      expiresIn: PROFILE_TOKEN_EXPIRES_IN,
    });
  };

  // SDK is the global source of truth. Changing it also keeps the persisted
  // onboarding language in sync (so the manual-path command builders and the
  // selection-confirmed gate stay coherent) and reports it as a language
  // selection to analytics. Ruby has no command-builder language, so the
  // persisted language is left as-is.
  const handleSdkChange = (nextSdk: Sdk) => {
    setSdk(nextSdk);
    const nextLanguage = sdkToWorkflowLanguage(nextSdk);
    if (nextLanguage) {
      setStoredOnboarding((prev: unknown) =>
        applyLanguageChange(normalizeOnboardingState(prev), nextLanguage),
      );
    }
    capture('onboarding_language_selected', {
      tenant_id: tenantId,
      user_email: currentUser?.email,
      language: nextSdk,
      source: 'onboarding_modal',
    });
  };

  const handleUseCaseChange = (nextUseCase: AvailableUseCaseKey) => {
    setStoredOnboarding((prev: unknown) =>
      applyUseCaseChange(normalizeOnboardingState(prev), nextUseCase),
    );
    capture('onboarding_use_case_selected', {
      tenant_id: tenantId,
      user_email: currentUser?.email,
      use_case: nextUseCase,
      source: 'onboarding_modal',
    });
  };

  // Fire once per open.
  useEffect(() => {
    if (open && !hasCapturedOpen.current) {
      hasCapturedOpen.current = true;
      capture('onboarding_modal_opened', {
        tenant_id: tenantId,
        user_email: currentUser?.email,
      });
    }
    if (!open) {
      hasCapturedOpen.current = false;
    }
  }, [open, capture, tenantId, currentUser?.email]);

  // Close on Esc and keep focus inside the overlay while it is open.
  useEffect(() => {
    if (!open) {
      return;
    }

    const container = containerRef.current;
    container?.focus();

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        onClose();
        return;
      }

      if (e.key !== 'Tab' || !container) {
        return;
      }

      const focusable = Array.from(
        container.querySelectorAll<HTMLElement>(
          'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])',
        ),
      );
      if (focusable.length === 0) {
        return;
      }

      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [open, onClose]);

  if (!open) {
    return null;
  }

  return (
    <div
      ref={containerRef}
      role="dialog"
      aria-modal="true"
      aria-label="Get started with Hatchet"
      tabIndex={-1}
      className="absolute inset-0 z-[110] overflow-y-auto bg-background outline-none"
    >
      {/* Center the content vertically within the region below the nav, while
          still allowing it to scroll when it is taller than the viewport. */}
      <div className="flex min-h-full items-center justify-center p-6">
        <div className="flex w-full max-w-4xl flex-col gap-6">
          <div className="space-y-1 text-center">
            <h2 className="text-2xl font-semibold tracking-tight">
              Get started with Hatchet
            </h2>
            <p className="text-sm text-muted-foreground">
              Set up your first worker and run your first workflow to get
              started with Hatchet.
            </p>
          </div>

          <OnboardingSteps
            tenantName={tenant?.name}
            sdk={sdk}
            onSdkChange={handleSdkChange}
            useCase={useCase}
            onUseCaseChange={handleUseCaseChange}
            onConfirmSelection={confirmSelection}
            profileToken={profileToken}
            isGeneratingProfileToken={createProfileTokenMutation.isPending}
            profileTokenError={profileTokenError}
            onGenerateProfileToken={handleGenerateProfileToken}
            canGenerateToken={canWrite}
            authDisabled={authDisabled}
            authDisabledToken={authDisabledToken}
            progress={progress}
            // Finish just closes the overlay; completion is derived from
            // useOnboardingProgress, never from a button.
            onFinish={onClose}
            onPromptGenerated={(template, promptSdk) => {
              capture('onboarding_prompt_generated', {
                tenant_id: tenantId,
                user_email: currentUser?.email,
                template,
                sdk: promptSdk,
                source: 'onboarding_modal',
              });
            }}
            onStepChangeEvent={(_step, stepLabel) => {
              capture('onboarding_tab_changed', {
                tenant_id: tenantId,
                user_email: currentUser?.email,
                tab: stepLabel,
                source: 'onboarding_modal',
              });
            }}
          />

          <div className="flex justify-center">
            <Button
              variant="ghost"
              size="sm"
              className="text-muted-foreground"
              onClick={onClose}
              hoverText="Your progress is saved. You can reopen this anytime from the overview page."
            >
              Skip onboarding
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
