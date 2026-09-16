import {
  LearnWorkflowSection,
  type InstallMethod,
  type WorkflowLanguageKey,
  type WorkflowStepKey,
  installMethodOptions,
  workflowStepOptions,
} from './learn-workflow-section';
import {
  applyLanguageChange,
  applyTabChange,
  applyUseCaseChange,
  normalizeOnboardingState,
  onboardingStorageKey,
} from './onboarding-state';
import { type AvailableUseCaseKey } from './use-case-options';
import { useOnboardingProgress } from './use-onboarding-progress';
import { SdkSwitcher, usePreferredSdk } from './use-preferred-sdk';
import { Button } from '@/components/v1/ui/button';
import { useAnalytics } from '@/hooks/use-analytics';
import useAuthDisabled from '@/hooks/use-auth-disabled';
import { useCurrentUser } from '@/hooks/use-current-user';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { useTenantDetails } from '@/hooks/use-tenant';
import api, { CreateAPITokenRequest } from '@/lib/api';
import { globalEmitter } from '@/lib/global-emitter';
import { cn } from '@/lib/utils';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { useMutation } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { useEffect, useMemo, useRef, useState } from 'react';

// A 100-year expiry, matching the profile-token flow on the Overview page.
const PROFILE_TOKEN_EXPIRES_IN = `${100 * 365 * 24 * 60 * 60}s`;

// Emit from anywhere (e.g. the Overview banner/footer) to open the modal.
export function openOnboarding() {
  globalEmitter.emit('open-onboarding', {});
}

// Full-screen onboarding overlay. It deliberately does NOT use the Radix
// Dialog: that overlay is z-[200] and would cover the top nav. Instead this
// fills the region below the 64px header (which is z-50) so the tenant
// switcher stays visible and interactive.
//
// This increment reuses the existing LearnWorkflowSection verbatim as the
// body to keep the modal functional. The redesigned steps and the
// agent/manual path fork land in the next increment; the step rail here is
// intentionally a lightweight indicator, not the final stepper.
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
  const navigate = useNavigate();
  const { capture } = useAnalytics();

  const [sdk, setSdk] = usePreferredSdk();

  const [installMethod, setInstallMethod] = useState<InstallMethod>(
    installMethodOptions.native.value,
  );
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

  const selectedTab: WorkflowStepKey = onboarding.tab;
  const language: WorkflowLanguageKey = onboarding.language;
  const useCase: AvailableUseCaseKey = onboarding.useCase;

  const setSelectedTab = (tab: WorkflowStepKey) =>
    setStoredOnboarding((prev: unknown) =>
      applyTabChange(
        normalizeOnboardingState(prev),
        tab,
        new Date().toISOString(),
      ),
    );

  const { workerConnected, runCompleted } = useOnboardingProgress(
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

  const steps = Object.values(workflowStepOptions);

  return (
    <div
      ref={containerRef}
      role="dialog"
      aria-modal="true"
      aria-label="Get started with Hatchet"
      tabIndex={-1}
      className="fixed inset-x-0 top-16 bottom-0 z-40 overflow-y-auto bg-background outline-none"
    >
      <div className="mx-auto flex w-full max-w-4xl flex-col gap-6 p-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="space-y-1">
            <h2 className="text-2xl font-semibold tracking-tight">
              Get started with Hatchet
            </h2>
            <p className="text-sm text-muted-foreground">
              Set up your first worker and run your first workflow to get
              started with Hatchet.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-4">
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted-foreground">SDK</span>
              <SdkSwitcher value={sdk} onChange={setSdk} />
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={onClose}
              aria-label="Exit onboarding"
              hoverText="Your progress is saved. You can reopen this anytime from the overview page."
            >
              Exit
            </Button>
          </div>
        </div>

        {/* Lightweight step rail. The next increment replaces this with the
            redesigned stepper and the agent/manual path fork. */}
        <ol className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
          {steps.map((step, index) => (
            <li
              key={step.value}
              className={cn(
                'font-medium',
                step.value === selectedTab && 'text-foreground',
              )}
            >
              {index + 1}. {step.label}
            </li>
          ))}
        </ol>

        {/* Reused as-is for this increment. Its own tab nav still drives the
            steps; the redesigned steps land next. */}
        <LearnWorkflowSection
          tenantName={tenant?.name}
          selectedTab={selectedTab}
          onSelectedTabChange={setSelectedTab}
          useCase={useCase}
          onUseCaseChange={(nextUseCase) => {
            setStoredOnboarding((prev: unknown) =>
              applyUseCaseChange(normalizeOnboardingState(prev), nextUseCase),
            );
          }}
          language={language}
          onLanguageChange={(nextLanguage) => {
            setStoredOnboarding((prev: unknown) =>
              applyLanguageChange(normalizeOnboardingState(prev), nextLanguage),
            );
          }}
          installMethod={installMethod}
          onInstallMethodChange={setInstallMethod}
          authDisabled={authDisabled}
          authDisabledToken={authDisabledToken}
          profileToken={profileToken}
          isGeneratingProfileToken={createProfileTokenMutation.isPending}
          profileTokenError={profileTokenError}
          onGenerateProfileToken={handleGenerateProfileToken}
          hasConnectedWorker={workerConnected}
          hasQualifiedRun={runCompleted}
          onViewRuns={() => {
            if (tenantId) {
              navigate({
                to: '/tenants/$tenant/runs',
                params: { tenant: tenantId },
              });
            }
            onClose();
          }}
          // Exit and Finish both just close the overlay in this increment;
          // closing never marks onboarding complete. Completion is derived
          // from useOnboardingProgress, not from a button.
          onSkip={onClose}
          onFinish={onClose}
          onTabChangeEvent={(_tab, tabLabel) => {
            capture('onboarding_tab_changed', {
              tenant_id: tenantId,
              user_email: currentUser?.email,
              tab: tabLabel,
              source: 'onboarding_modal',
            });
          }}
          onLanguageSelectedEvent={(_language, languageLabel) => {
            capture('onboarding_language_selected', {
              tenant_id: tenantId,
              user_email: currentUser?.email,
              language: languageLabel,
              source: 'onboarding_modal',
            });
          }}
          onUseCaseSelectedEvent={(useCaseKey, useCaseLabel) => {
            capture('onboarding_use_case_selected', {
              tenant_id: tenantId,
              user_email: currentUser?.email,
              use_case: useCaseKey,
              use_case_label: useCaseLabel,
              source: 'onboarding_modal',
            });
          }}
        />
      </div>
    </div>
  );
}
