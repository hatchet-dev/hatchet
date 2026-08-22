import { CreateApiTokenSection } from './components/create-api-token-section';
import { FinishOnboardingDialog } from './components/finish-onboarding-dialog';
import {
  LearnWorkflowSection,
  type WorkflowLanguageKey,
  type WorkflowStepKey,
  type InstallMethod,
  installMethodOptions,
  workflowStepOptions,
} from './components/learn-workflow-section';
import {
  applyLanguageChange,
  applyTabChange,
  applyUseCaseChange,
  hasQualifiedWorker,
  normalizeOnboardingState,
  onboardingStorageKey,
  qualifiedRunQueryParams,
  type OnboardingPersistedState,
} from './components/onboarding-state';
import { SkipOnboardingDialog } from './components/skip-onboarding-dialog';
import { SupportSection } from './components/support-section';
import { TokenSuccessDialog } from './components/token-success-dialog';
import { type AvailableUseCaseKey } from './components/use-case-options';
import { useAnalytics } from '@/hooks/use-analytics';
import useAuthDisabled from '@/hooks/use-auth-disabled';
import useControlPlane from '@/hooks/use-control-plane';
import { useCurrentUser } from '@/hooks/use-current-user';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { useTenantDetails } from '@/hooks/use-tenant';
import api, { CreateAPITokenRequest, queries } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { useEffect, useMemo, useRef, useState } from 'react';

const EXPIRES_IN_OPTIONS = {
  '3 months': `${3 * 30 * 24 * 60 * 60}s`,
  '1 year': `${365 * 24 * 60 * 60}s`,
  '100 years': `${100 * 365 * 24 * 60 * 60}s`,
};

export default function Overview() {
  const { tenant, tenantId } = useTenantDetails();
  const { currentUser } = useCurrentUser();
  const authDisabled = useAuthDisabled();
  const { meta } = useApiMeta();
  const authDisabledToken =
    meta && 'authDisabledToken' in meta ? meta.authDisabledToken : undefined;
  const navigate = useNavigate();
  const { capture } = useAnalytics();
  const [tokenName, setTokenName] = useState('');
  const [hasEditedTokenName, setHasEditedTokenName] = useState(false);
  const [expiresIn, setExpiresIn] = useState(EXPIRES_IN_OPTIONS['100 years']);
  const [generatedToken, setGeneratedToken] = useState<string | undefined>();
  const [showTokenDialog, setShowTokenDialog] = useState(false);
  const [showFinishDialog, setShowFinishDialog] = useState(false);
  const [showSkipDialog, setShowSkipDialog] = useState(false);
  const [profileToken, setProfileToken] = useState<string | undefined>();
  const [profileTokenError, setProfileTokenError] = useState<
    string | undefined
  >();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [installMethod, setInstallMethod] = useState<InstallMethod>(
    installMethodOptions.native.value,
  );
  const hasTrackedWorkerConnection = useRef(false);
  const { isSelfHosted } = useControlPlane();

  // The raw stored value is normalized on every read, so malformed or
  // stale entries fall back to the defaults.
  const [storedOnboarding, setStoredOnboarding] = useLocalStorageState<unknown>(
    onboardingStorageKey(tenantId ?? 'unknown'),
    null,
  );
  const onboarding = useMemo(
    () => normalizeOnboardingState(storedOnboarding),
    [storedOnboarding],
  );
  const updateOnboarding = (patch: Partial<OnboardingPersistedState>) => {
    setStoredOnboarding((prev: unknown) => ({
      ...normalizeOnboardingState(prev),
      ...patch,
    }));
  };

  const selectedTab: WorkflowStepKey = onboarding.tab;
  // All tab navigation, including direct tab clicks and Continue buttons,
  // goes through applyTabChange so leaving Choose use case always confirms
  // the selection.
  const setSelectedTab = (tab: WorkflowStepKey) =>
    setStoredOnboarding((prev: unknown) =>
      applyTabChange(
        normalizeOnboardingState(prev),
        tab,
        new Date().toISOString(),
      ),
    );
  const language: WorkflowLanguageKey = onboarding.language;
  const useCase: AvailableUseCaseKey = onboarding.useCase;

  const defaultTokenName = useMemo(() => {
    const name = currentUser?.name?.trim();
    if (!name) {
      return '';
    }

    return `${name}'s token`;
  }, [currentUser?.name]);

  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  useEffect(() => {
    if (hasEditedTokenName) {
      return;
    }

    if (tokenName.trim()) {
      return;
    }

    if (!defaultTokenName) {
      return;
    }

    setTokenName(defaultTokenName);
    setFieldErrors((prev) => (prev.name ? {} : prev));
  }, [defaultTokenName, hasEditedTokenName, tokenName]);

  // Track page view on mount
  useEffect(() => {
    capture('onboarding_overview_viewed', {
      tenant_id: tenantId,
      user_email: currentUser?.email,
    });
  }, [capture, tenantId, currentUser?.email]);

  const createTokenMutation = useMutation({
    mutationKey: ['api-token:create', tenantId],
    mutationFn: async (data: CreateAPITokenRequest) => {
      const res = await api.apiTokenCreate(tenantId!, data);
      return res.data;
    },
    onSuccess: (data) => {
      setGeneratedToken(data.token);
      setShowTokenDialog(true);
      // Track token generation
      capture('onboarding_token_generated', {
        tenant_id: tenantId,
        user_email: currentUser?.email,
        token_name: tokenName,
        expires_in: expiresIn,
      });
      // Reset form
      setHasEditedTokenName(false);
      setTokenName('');
    },
    onError: handleApiError,
  });

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
        expires_in: EXPIRES_IN_OPTIONS['100 years'],
        source: 'learn_workflow_profile_step',
      });
    },
    onError: () => {
      setProfileTokenError('Failed to generate token. Please try again.');
    },
  });

  const handleGenerateToken = () => {
    if (!tokenName.trim()) {
      setFieldErrors({ name: 'Name is required' });
      return;
    }
    createTokenMutation.mutate({
      name: tokenName,
      expiresIn: expiresIn,
    });
  };

  const handleGenerateProfileToken = () => {
    setProfileTokenError(undefined);
    createProfileTokenMutation.mutate({
      name: defaultTokenName ? `${defaultTokenName} (CLI)` : 'CLI token',
      expiresIn: EXPIRES_IN_OPTIONS['100 years'],
    });
  };

  const selectionConfirmedAt = onboarding.selectionConfirmedAt;

  // Poll for workers while "Project quickstart" is visible and onboarding
  // is shown. Polling keeps running there even after a worker qualifies,
  // because the indicator must also flip back when a worker disconnects.
  // hasQualifiedWorker decides the connected state; the rows alone
  // include workers that are stale or from a previous selection.
  const workersQuery = useQuery({
    ...queries.workers.list(tenantId!),
    enabled:
      !!tenantId &&
      !onboarding.hidden &&
      selectedTab === workflowStepOptions.quickstart.value,
    refetchInterval: 2000, // Poll every 2 seconds
  });

  const hasConnectedWorker = hasQualifiedWorker(
    workersQuery.data?.rows ?? [],
    selectionConfirmedAt,
  );

  // Detect a completed run created after the confirmed selection. Worker
  // qualification above is current state and reverts on disconnect; a
  // completed run is historical product state and stays complete while
  // the confirmation timestamp stands. Both reset together when a
  // selection change or Restart onboarding clears the timestamp. The
  // query runs on any tab whenever a timestamp exists, so completion
  // survives a reload onto Finish, and it polls only while the "Run a
  // task" tab is waiting for a result. The timestamp is part of the query
  // key, so a reset selection never reuses stale completion data. Runs
  // are tenant-scoped, so a teammate's completed run after the
  // confirmation also satisfies this check.
  const qualifiedRunQuery = useQuery({
    ...queries.v1WorkflowRuns.list(
      tenantId!,
      qualifiedRunQueryParams(
        selectionConfirmedAt ?? new Date(0).toISOString(),
      ),
      isSelfHosted,
    ),
    enabled: !!tenantId && !!selectionConfirmedAt && !onboarding.hidden,
    refetchInterval: (query) => {
      const hasRows =
        query.state.data !== 'timeout' &&
        (query.state.data?.rows?.length ?? 0) > 0;
      return selectedTab === workflowStepOptions.runTask.value && !hasRows
        ? 2000
        : false;
    },
  });

  const hasQualifiedRun =
    !!selectionConfirmedAt &&
    qualifiedRunQuery.data !== 'timeout' &&
    (qualifiedRunQuery.data?.rows?.length ?? 0) > 0;

  // The connection event fires once for each tenant, so a tenant switch
  // without a remount must re-arm it.
  useEffect(() => {
    hasTrackedWorkerConnection.current = false;
  }, [tenantId]);

  // Track worker connection (only once)
  useEffect(() => {
    if (hasConnectedWorker && !hasTrackedWorkerConnection.current) {
      capture('onboarding_worker_connected', {
        tenant_id: tenantId,
        user_email: currentUser?.email,
      });
      hasTrackedWorkerConnection.current = true;
    }
  }, [hasConnectedWorker, capture, tenantId, currentUser?.email]);

  return (
    <div className="flex h-full w-full flex-col gap-y-8 lg:p-6">
      <div className="grid gap-2 grid-cols-1 items-start lg:grid-cols-[1fr_auto]">
        <div className="flex items-center gap-6 flex-wrap">
          <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
        </div>
      </div>

      {/* Hidden onboarding leaves no trace on Overview, whether hidden by
          Skip or by Finish; recovery is Restart onboarding on the tenant
          General settings page. */}
      {!onboarding.hidden && (
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
          hasConnectedWorker={hasConnectedWorker}
          hasQualifiedRun={hasQualifiedRun}
          onViewRuns={() => {
            navigate({
              to: '/tenants/$tenant/runs',
              params: { tenant: tenantId! },
            });
          }}
          onSkip={() => setShowSkipDialog(true)}
          onTabChangeEvent={(_tab, tabLabel) => {
            capture('onboarding_tab_changed', {
              tenant_id: tenantId,
              user_email: currentUser?.email,
              tab: tabLabel,
            });
          }}
          onLanguageSelectedEvent={(_language, languageLabel) => {
            capture('onboarding_language_selected', {
              tenant_id: tenantId,
              user_email: currentUser?.email,
              language: languageLabel,
            });
          }}
          onUseCaseSelectedEvent={(useCaseKey, useCaseLabel) => {
            capture('onboarding_use_case_selected', {
              tenant_id: tenantId,
              user_email: currentUser?.email,
              use_case: useCaseKey,
              use_case_label: useCaseLabel,
            });
          }}
          onFinish={() => setShowFinishDialog(true)}
        />
      )}

      {!authDisabled && !onboarding.hidden && (
        <CreateApiTokenSection
          tokenName={tokenName}
          onTokenNameChange={(value) => {
            setHasEditedTokenName(true);
            setTokenName(value);
            setFieldErrors({});
          }}
          expiresIn={expiresIn}
          expiresInOptions={EXPIRES_IN_OPTIONS}
          onExpiresInChange={setExpiresIn}
          onExpiresInSelected={(label) => {
            capture('onboarding_token_expiration_selected', {
              tenant_id: tenantId,
              user_email: currentUser?.email,
              expiration: label,
            });
          }}
          fieldErrors={fieldErrors}
          isGenerating={createTokenMutation.isPending}
          onGenerateToken={handleGenerateToken}
        />
      )}

      <SupportSection />

      <SkipOnboardingDialog
        open={showSkipDialog}
        onOpenChange={setShowSkipDialog}
        onConfirm={() => {
          updateOnboarding({ hidden: true });
          setShowSkipDialog(false);
        }}
      />

      <FinishOnboardingDialog
        open={showFinishDialog}
        onOpenChange={setShowFinishDialog}
        onConfirm={() => {
          // The confirmed OK is the completion action, so the event fires
          // exactly once here; opening or dismissing the dialog captures
          // nothing.
          capture('onboarding_completed', {
            tenant_id: tenantId,
            user_email: currentUser?.email,
          });
          updateOnboarding({ hidden: true });
          setShowFinishDialog(false);
          navigate({
            to: '/tenants/$tenant/runs',
            params: { tenant: tenantId! },
          });
        }}
      />

      <TokenSuccessDialog
        open={showTokenDialog}
        onOpenChange={setShowTokenDialog}
        token={generatedToken}
      />
    </div>
  );
}
