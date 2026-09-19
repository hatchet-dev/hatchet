import {
  applyUseCaseChange,
  normalizeOnboardingState,
  onboardingStorageKey,
} from './onboarding-state';
import {
  OnboardingSteps,
  type SetupPath,
  type StepKey,
} from './onboarding-steps';
import { type AgentPatternKey } from './prompt-templates';
import { type AvailableUseCaseKey } from './use-case-options';
import { useOnboardingProgress } from './use-onboarding-progress';
import { usePreferredSdk, type Sdk } from './use-preferred-sdk';
import { SetupCard } from '@/components/layout/setup-card';
import { Button } from '@/components/v1/ui/button';
import { useAnalytics } from '@/hooks/use-analytics';
import useAuthDisabled from '@/hooks/use-auth-disabled';
import useCanWrite from '@/hooks/use-can-write';
import { useCurrentUser } from '@/hooks/use-current-user';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { useTenantDetails } from '@/hooks/use-tenant';
import api, { CreateAPITokenRequest, queries } from '@/lib/api';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

// Query-key prefixes behind the Overview panels and its setup banner. The user
// usually creates their first worker and run while this overlay covers the
// Overview, so they are invalidated on close: the panels refetch in the
// background and show the new run instead of stale (often empty) data.
const OVERVIEW_DATA_KEY_PREFIXES = [
  'worker:list',
  'v1:workflow-run:list',
  'v1:task-run:metrics',
  'v1-task:metrics',
  'queue-metrics:get:step-run',
];

// A 100-year expiry: the token backs a long-lived local CLI profile.
const PROFILE_TOKEN_EXPIRES_IN = `${100 * 365 * 24 * 60 * 60}s`;

// The agent-path selections live under their own tenant-scoped key so a refresh
// restores them without widening the shared onboarding state schema.
type AgentSelections = {
  patterns?: AgentPatternKey[];
  description?: string;
};

const agentSelectionsKey = (tenantId: string) =>
  `hatchet:onboarding-agent:${tenantId}`;

// The onboarding overlay. It renders into AppLayout's overlay slot so it covers
// the page and the sidebar but not the header: the tenant switcher stays
// usable. z-[110] sits above the sidebar (z-[100]) and below Radix dialogs and
// popovers (z-[200]+).
//
// It is mounted only while the tenant onboarding route matches, keyed by
// tenant (see authenticated.tsx), so its polling stops on close and no state
// (such as a freshly generated token) leaks across a tenant switch.
export function OnboardingModal({
  onClose,
  path,
  step,
  onNavigate,
}: {
  onClose: () => void;
  path: SetupPath | null;
  step?: StepKey;
  onNavigate: (next: { path: SetupPath | null; step: StepKey }) => void;
}) {
  const { tenant, tenantId } = useTenantDetails();
  const { currentUser } = useCurrentUser();
  const authDisabled = useAuthDisabled();
  const { meta } = useApiMeta();
  const authDisabledToken =
    meta && 'authDisabledToken' in meta ? meta.authDisabledToken : undefined;
  const { capture } = useAnalytics();
  const canWrite = useCanWrite();
  const queryClient = useQueryClient();

  const [sdk, setSdk] = usePreferredSdk();

  const [profileToken, setProfileToken] = useState<string | undefined>();
  const [profileTokenError, setProfileTokenError] = useState<
    string | undefined
  >();

  const containerRef = useRef<HTMLDivElement>(null);

  const [storedOnboarding, setStoredOnboarding] = useLocalStorageState<unknown>(
    onboardingStorageKey(tenantId ?? 'unknown'),
    null,
  );
  const onboarding = useMemo(
    () => normalizeOnboardingState(storedOnboarding),
    [storedOnboarding],
  );

  const [agentSelections, setAgentSelections] =
    useLocalStorageState<AgentSelections>(
      agentSelectionsKey(tenantId ?? 'unknown'),
      {},
    );

  // Progress only counts workers and runs created after the selection was
  // confirmed. Changing the template clears the timestamp (applyUseCaseChange),
  // so this sets it whenever it is missing rather than only once.
  const confirmSelection = () =>
    setStoredOnboarding((prev: unknown) => {
      const state = normalizeOnboardingState(prev);
      return state.selectionConfirmedAt
        ? state
        : { ...state, selectionConfirmedAt: new Date().toISOString() };
    });

  const progress = useOnboardingProgress(
    tenantId,
    onboarding.selectionConfirmedAt ?? undefined,
  );

  const tokensQuery = useQuery({
    ...queries.tokens.list(tenantId ?? ''),
    enabled: !!tenantId,
    // Poll (for tokens created elsewhere, e.g. the settings page) only until
    // one exists.
    refetchInterval: (query) =>
      (query.state.data?.rows?.length ?? 0) > 0 ? false : 2000,
  });
  // Checked against the API so the gate survives a refresh. A token generated
  // here counts immediately, without waiting for the list to refetch, and an
  // auth-disabled instance needs no token at all.
  const hasApiToken =
    (tokensQuery.data?.rows?.length ?? 0) > 0 || !!profileToken || authDisabled;

  const handleClose = useCallback(() => {
    OVERVIEW_DATA_KEY_PREFIXES.forEach((prefix) => {
      void queryClient.invalidateQueries({ queryKey: [prefix] });
    });
    onClose();
  }, [queryClient, onClose]);

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
      void queryClient.invalidateQueries({ queryKey: ['api-token:list'] });
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

  const handleSdkChange = (nextSdk: Sdk) => {
    setSdk(nextSdk);
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

  // The ref keeps this to one event per open even if the user's email resolves
  // after mount and re-runs the effect.
  const hasCapturedOpen = useRef(false);
  useEffect(() => {
    if (hasCapturedOpen.current) {
      return;
    }
    hasCapturedOpen.current = true;
    capture('onboarding_modal_opened', {
      tenant_id: tenantId,
      user_email: currentUser?.email,
    });
  }, [capture, tenantId, currentUser?.email]);

  // Focus moves into the overlay once, on mount. It deliberately does not trap
  // focus, because the header stays interactive during the flow.
  useEffect(() => {
    containerRef.current?.focus();
  }, []);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Radix layers (the help menu, the tenant switcher) handle Escape in the
      // capture phase and mark it prevented; closing the whole flow as well
      // would be a surprise.
      if (e.key !== 'Escape' || e.defaultPrevented) {
        return;
      }
      e.preventDefault();
      handleClose();
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleClose]);

  return (
    <div
      ref={containerRef}
      // Not aria-modal: the header above stays interactive, so the rest of the
      // page is made inert by AppLayout instead.
      role="dialog"
      aria-label="Run your first task"
      tabIndex={-1}
      className="absolute inset-0 z-[110] overflow-y-auto bg-background outline-none"
    >
      <div className="flex min-h-full items-center justify-center p-6">
        <div className="flex w-full flex-col items-center">
          <SetupCard
            title="Run your first task"
            description="Connect a locally running worker and run your first task."
            className="max-w-3xl"
          >
            <OnboardingSteps
              tenantName={tenant?.name}
              tenantId={tenantId}
              path={path}
              step={step}
              onNavigate={onNavigate}
              patterns={agentSelections.patterns ?? []}
              onPatternsChange={(next) =>
                setAgentSelections((prev) => ({ ...prev, patterns: next }))
              }
              description={agentSelections.description ?? ''}
              onDescriptionChange={(next) =>
                setAgentSelections((prev) => ({ ...prev, description: next }))
              }
              sdk={sdk}
              onSdkChange={handleSdkChange}
              useCase={onboarding.useCase}
              onUseCaseChange={handleUseCaseChange}
              onConfirmSelection={confirmSelection}
              profileToken={profileToken}
              isGeneratingProfileToken={createProfileTokenMutation.isPending}
              profileTokenError={profileTokenError}
              onGenerateProfileToken={handleGenerateProfileToken}
              canGenerateToken={canWrite}
              hasApiToken={hasApiToken}
              authDisabled={authDisabled}
              authDisabledToken={authDisabledToken}
              progress={progress}
              onFinish={handleClose}
              onPromptGenerated={(promptPatterns, promptSdk) => {
                capture('onboarding_prompt_generated', {
                  tenant_id: tenantId,
                  user_email: currentUser?.email,
                  patterns: promptPatterns,
                  sdk: promptSdk,
                  source: 'onboarding_modal',
                });
              }}
              onStepChangeEvent={(stepLabel) => {
                capture('onboarding_tab_changed', {
                  tenant_id: tenantId,
                  user_email: currentUser?.email,
                  tab: stepLabel,
                  source: 'onboarding_modal',
                });
              }}
            />
          </SetupCard>

          <div className="mt-4 flex justify-center">
            <Button
              variant="ghost"
              size="sm"
              className="text-xs text-muted-foreground"
              onClick={handleClose}
              hoverText="Your progress is saved. You can reopen this anytime from the overview page."
            >
              Skip for now
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
