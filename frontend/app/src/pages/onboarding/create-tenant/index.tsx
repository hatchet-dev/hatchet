import { NewTenantSaverForm } from '@/components/forms/new-tenant-saver-form';
import { SetupCard, SetupScreen } from '@/components/layout/setup-card';
import { Button } from '@/components/v1/ui/button';
import { queries } from '@/lib/api';
import { useUserApi } from '@/lib/api/user-wrapper';
import { useRedirectOrNavigate } from '@/lib/redirect';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { useMutation } from '@tanstack/react-query';
import { useLoaderData, useNavigate } from '@tanstack/react-router';

export default function CreateTenant() {
  const redirectOrNavigate = useRedirectOrNavigate();
  const navigate = useNavigate();
  const { userUpdateLogoutMutation } = useUserApi();
  const { organizations } = useLoaderData({
    from: '/onboarding/create-tenant',
  });

  const logoutMutation = useMutation({
    ...userUpdateLogoutMutation(),
    onSettled: () => {
      queryClient.clear();
      navigate({ to: appRoutes.authLoginRoute.to });
    },
  });

  const defaultOrganizationId =
    organizations && organizations.length > 0
      ? organizations[0].metadata.id
      : undefined;

  return (
    <SetupScreen
      topRight={
        <Button
          variant="ghost"
          size="sm"
          onClick={() => logoutMutation.mutate()}
          disabled={logoutMutation.isPending}
        >
          Sign out
        </Button>
      }
    >
      <SetupCard
        title="Create a new tenant"
        description="A tenant is an isolated environment for your workflows. Set one up to get started."
      >
        <NewTenantSaverForm
          defaultOrganizationId={defaultOrganizationId}
          afterSave={(result) => {
            const tenantId =
              result.type === 'cloud'
                ? result.tenant.id
                : result.tenant.metadata.id;

            if (result.type === 'cloud') {
              void queryClient
                .prefetchQuery(queries.controlPlane.subscriptionPlans())
                .catch(() => {
                  // Ignore prefetch errors; subscription plans will be fetched on demand if needed.
                });
            }

            redirectOrNavigate({
              to: appRoutes.tenantOverviewRoute.to,
              params: { tenant: tenantId },
              replace: true,
            });
          }}
        />
      </SetupCard>
    </SetupScreen>
  );
}
