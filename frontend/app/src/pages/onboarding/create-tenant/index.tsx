import { NewTenantSaverForm } from '@/components/forms/new-tenant-saver-form';
import { Button } from '@/components/v1/ui/button';
import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';
import { queries } from '@/lib/api';
import { useUserApi } from '@/lib/api/user-wrapper';
import { useRedirectOrNavigate } from '@/lib/redirect';
import { AuthLayout } from '@/pages/auth/components/auth-layout';
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
    <div className="fixed inset-0 z-50 bg-background">
      <div className="absolute top-4 right-4 z-10">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => logoutMutation.mutate()}
          disabled={logoutMutation.isPending}
        >
          Sign out
        </Button>
      </div>
      <AuthLayout>
        <div className="flex flex-col gap-3 text-center lg:text-left w-full">
          <div className="flex justify-center pb-3 lg:hidden">
            <HatchetLogo className="h-8 w-auto" />
          </div>
          <h2 className="text-2xl font-semibold tracking-tight">
            Create a new tenant
          </h2>
          <p className="text-sm text-muted-foreground">
            A tenant is an isolated environment for your workflows. Set one up
            to get started.
          </p>
        </div>

        <NewTenantSaverForm
          defaultOrganizationId={defaultOrganizationId}
          afterSave={(result) => {
            const tenantId =
              result.type === 'cloud'
                ? result.tenant.id
                : result.tenant.metadata.id;

            if (result.type === 'cloud') {
              void queryClient
                .prefetchQuery(queries.cloud.subscriptionPlans())
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
      </AuthLayout>
    </div>
  );
}
