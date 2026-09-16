import { CreateApiTokenSection } from '../create-api-token-section';
import { TokenSuccessDialog } from '../token-success-dialog';
import { PanelCard, PanelState } from './panel-card';
import { Button } from '@/components/v1/ui/button';
import { SecretCopier } from '@/components/v1/ui/secret-copier';
import { useCurrentUser } from '@/hooks/use-current-user';
import api, { type CreateAPITokenRequest, queries } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { appRoutes } from '@/router';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { KeyRound } from 'lucide-react';
import { useMemo, useState } from 'react';

const EXPIRES_IN_OPTIONS = {
  '3 months': `${3 * 30 * 24 * 60 * 60}s`,
  '1 year': `${365 * 24 * 60 * 60}s`,
  '100 years': `${100 * 365 * 24 * 60 * 60}s`,
};

export function SetupPanel({
  tenantId,
  authDisabled,
  authDisabledToken,
}: {
  tenantId: string;
  authDisabled: boolean;
  authDisabledToken?: string;
}) {
  const { currentUser } = useCurrentUser();
  const [tokenName, setTokenName] = useState('');
  const [hasEditedTokenName, setHasEditedTokenName] = useState(false);
  const [expiresIn, setExpiresIn] = useState(EXPIRES_IN_OPTIONS['100 years']);
  const [generatedToken, setGeneratedToken] = useState<string | undefined>();
  const [showTokenDialog, setShowTokenDialog] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const { handleApiError } = useApiError({ setFieldErrors });

  const defaultTokenName = useMemo(() => {
    const name = currentUser?.name?.trim();
    return name ? `${name}'s token` : '';
  }, [currentUser?.name]);

  const resolvedTokenName =
    tokenName || (!hasEditedTokenName ? defaultTokenName : '');

  const tokensQuery = useQuery({
    ...queries.tokens.list(tenantId),
    enabled: !authDisabled,
  });

  const hasToken = (tokensQuery.data?.rows?.length ?? 0) > 0;

  const createTokenMutation = useMutation({
    mutationKey: ['api-token:create', tenantId],
    mutationFn: async (data: CreateAPITokenRequest) => {
      const res = await api.apiTokenCreate(tenantId, data);
      return res.data;
    },
    onSuccess: (data) => {
      setGeneratedToken(data.token);
      setShowTokenDialog(true);
      setHasEditedTokenName(false);
      setTokenName('');
      void tokensQuery.refetch();
    },
    onError: handleApiError,
  });

  const handleGenerateToken = () => {
    if (!resolvedTokenName.trim()) {
      setFieldErrors({ name: 'Name is required' });
      return;
    }
    createTokenMutation.mutate({
      name: resolvedTokenName,
      expiresIn,
    });
  };

  return (
    <PanelCard icon={<KeyRound className="size-4" />} title="Setup">
      {authDisabled ? (
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Auth is disabled on this instance. Use the built-in token below.
          </p>
          {authDisabledToken && (
            <SecretCopier
              secrets={{ HATCHET_CLIENT_TOKEN: authDisabledToken }}
              className="text-sm"
              copy
            />
          )}
        </div>
      ) : (
        <PanelState
          loading={tokensQuery.isLoading}
          error={tokensQuery.isError}
          onRetry={() => tokensQuery.refetch()}
          isEmpty={false}
          emptyText=""
        >
          {hasToken ? (
            <div className="space-y-3">
              <div className="space-y-1">
                <p className="text-sm font-medium">API tokens</p>
                <p className="text-sm text-muted-foreground">
                  You have at least one active token.
                </p>
              </div>
              <Button variant="outline" size="sm" asChild>
                <Link
                  to={appRoutes.tenantSettingsApiTokensRoute.to}
                  params={{ tenant: tenantId }}
                >
                  Manage API tokens
                </Link>
              </Button>
            </div>
          ) : (
            <div className="space-y-3">
              <p className="text-sm text-muted-foreground">
                You need a token for workers and scripts to authenticate.
              </p>
              <CreateApiTokenSection
                tokenName={resolvedTokenName}
                onTokenNameChange={(value) => {
                  setHasEditedTokenName(true);
                  setTokenName(value);
                  setFieldErrors({});
                }}
                expiresIn={expiresIn}
                expiresInOptions={EXPIRES_IN_OPTIONS}
                onExpiresInChange={setExpiresIn}
                fieldErrors={fieldErrors}
                isGenerating={createTokenMutation.isPending}
                onGenerateToken={handleGenerateToken}
              />
            </div>
          )}
        </PanelState>
      )}

      <TokenSuccessDialog
        open={showTokenDialog}
        onOpenChange={setShowTokenDialog}
        token={generatedToken}
      />
    </PanelCard>
  );
}
