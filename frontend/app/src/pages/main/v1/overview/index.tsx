import { CreateApiTokenSection } from './components/create-api-token-section';
import {
  LearnWorkflowSection,
  type WorkflowLanguageKey,
  type WorkflowStepKey,
  type InstallMethod,
  workflowLanguageOptions,
  installMethodOptions,
  workflowStepOptions,
} from './components/learn-workflow-section';
import { SupportSection } from './components/support-section';
import { TokenSuccessDialog } from './components/token-success-dialog';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';
import { Spinner } from '@/components/v1/ui/loading';
import { useAnalytics } from '@/hooks/use-analytics';
import { useCurrentUser } from '@/hooks/use-current-user';
import { useTenantDetails } from '@/hooks/use-tenant';
import api, { CreateAPITokenRequest, queries } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { appRoutes } from '@/router';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate, useSearch } from '@tanstack/react-router';
import { useEffect, useMemo, useRef, useState } from 'react';

const EXPIRES_IN_OPTIONS = {
  '3 months': `${3 * 30 * 24 * 60 * 60}s`,
  '1 year': `${365 * 24 * 60 * 60}s`,
  '100 years': `${100 * 365 * 24 * 60 * 60}s`,
};

export default function Overview() {
  const { tenant, tenantId } = useTenantDetails();
  const { currentUser } = useCurrentUser();
  const navigate = useNavigate();
  const { capture } = useAnalytics();
  const search = useSearch({ strict: false }) as { welcome?: boolean };
  const [showWelcome, setShowWelcome] = useState(!!search.welcome);

  const plansQuery = useQuery({
    ...queries.cloud.subscriptionPlans(),
    enabled: showWelcome,
  });
  const freeLimits = plansQuery.data?.freeLimits;
  const [tokenName, setTokenName] = useState('');
  const [hasEditedTokenName, setHasEditedTokenName] = useState(false);
  const [expiresIn, setExpiresIn] = useState(EXPIRES_IN_OPTIONS['100 years']);
  const [generatedToken, setGeneratedToken] = useState<string | undefined>();
  const [showTokenDialog, setShowTokenDialog] = useState(false);
  const [profileToken, setProfileToken] = useState<string | undefined>();
  const [profileTokenError, setProfileTokenError] = useState<
    string | undefined
  >();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [selectedTab, setSelectedTab] = useState<WorkflowStepKey>(
    workflowStepOptions.install.value,
  );
  const [language, setLanguage] = useState<WorkflowLanguageKey>(
    workflowLanguageOptions.python.value,
  );
  const [installMethod, setInstallMethod] = useState<InstallMethod>(
    installMethodOptions.native.value,
  );
  const hasTrackedWorkerConnection = useRef(false);

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

  // Poll for workers when on the "Run worker" tab
  const workersQuery = useQuery({
    ...queries.workers.list(tenantId!),
    enabled: selectedTab === workflowStepOptions.quickstart.value,
    refetchInterval: 2000, // Poll every 2 seconds
  });

  const hasActiveWorker = (workersQuery.data?.rows?.length ?? 0) > 0;

  // Track worker connection (only once)
  useEffect(() => {
    if (hasActiveWorker && !hasTrackedWorkerConnection.current) {
      capture('onboarding_worker_connected', {
        tenant_id: tenantId,
        user_email: currentUser?.email,
      });
      hasTrackedWorkerConnection.current = true;
    }
  }, [hasActiveWorker, capture, tenantId, currentUser?.email]);

  return (
    <div className="flex h-full w-full flex-col gap-y-8 lg:p-6">
      <div className="grid gap-2 grid-cols-1 items-start lg:grid-cols-[1fr_auto]">
        <div className="flex items-center gap-6 flex-wrap">
          <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
        </div>
      </div>

      <LearnWorkflowSection
        tenantName={tenant?.name}
        selectedTab={selectedTab}
        onSelectedTabChange={setSelectedTab}
        language={language}
        onLanguageChange={setLanguage}
        installMethod={installMethod}
        onInstallMethodChange={setInstallMethod}
        profileToken={profileToken}
        isGeneratingProfileToken={createProfileTokenMutation.isPending}
        profileTokenError={profileTokenError}
        onGenerateProfileToken={handleGenerateProfileToken}
        hasActiveWorker={hasActiveWorker}
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
        onFinish={() => {
          capture('onboarding_completed', {
            tenant_id: tenantId,
            user_email: currentUser?.email,
          });
          navigate({
            to: '/tenants/$tenant/runs',
            params: { tenant: tenantId! },
          });
        }}
      />

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

      <SupportSection />

      <TokenSuccessDialog
        open={showTokenDialog}
        onOpenChange={setShowTokenDialog}
        token={generatedToken}
      />

      <Dialog
        open={showWelcome}
        onOpenChange={(open) => {
          if (!open) {
            setShowWelcome(false);
            navigate({
              to: appRoutes.tenantOverviewRoute.to,
              params: { tenant: tenantId! },
              search: {},
              replace: true,
            });
          }
        }}
      >
        <DialogContent className="max-w-md text-center">
          <div className="flex flex-col items-center gap-5">
            <HatchetLogo variant="mark" className="h-8 w-8" />
            <div className="space-y-2">
              <DialogTitle className="text-center text-xl">
                Welcome to Hatchet
              </DialogTitle>
              <DialogDescription className="text-center text-sm text-muted-foreground">
                You&apos;re on the free plan with generous limits to get
                started. We&apos;ll let you know when you&apos;re getting close.
              </DialogDescription>
            </div>
            <ul className="w-full text-left text-sm space-y-2 rounded-md border border-border/50 bg-muted/30 p-4">
              {plansQuery.isLoading ? (
                <li className="flex justify-center py-2">
                  <Spinner />
                </li>
              ) : (
                freeLimits?.map((fl) => (
                  <li key={fl.featureId} className="flex justify-between">
                    <span className="text-muted-foreground">{fl.name}</span>
                    <span className="font-medium">
                      {fl.limit.toLocaleString()}
                    </span>
                  </li>
                ))
              )}
            </ul>
            <p className="text-xs text-muted-foreground">
              You can upgrade anytime from Billing & Limits in your tenant
              settings.
            </p>
            <div className="flex w-full flex-col gap-2">
              <Button
                className="w-full"
                onClick={() => {
                  setShowWelcome(false);
                  navigate({
                    to: appRoutes.tenantOverviewRoute.to,
                    params: { tenant: tenantId! },
                    search: {},
                    replace: true,
                  });
                }}
              >
                Get Started
              </Button>
              <Button
                variant="ghost"
                className="w-full"
                onClick={() => {
                  setShowWelcome(false);
                  navigate({
                    to: '/tenants/$tenant/tenant-settings/billing-and-limits',
                    params: { tenant: tenantId! },
                  });
                }}
              >
                Explore Plans
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
