import { NewOrganizationSaverForm } from '@/components/forms/new-organization-saver-form';
import { Button } from '@/components/v1/ui/button';
import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';
import { queries } from '@/lib/api';
import { useUserApi } from '@/lib/api/user-wrapper';
import freeEmailDomains from '@/lib/free-email-domains.json';
import { useRedirectOrNavigate } from '@/lib/redirect';
import { AuthLayout } from '@/pages/auth/components/auth-layout';
import { useAppContext } from '@/providers/app-context';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { useMutation } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';

const FREE_EMAIL_DOMAIN_SET = new Set(freeEmailDomains);

function deriveDefaultOrgName(user: { name?: string; email: string }): string {
  const domain = user.email.split('@')[1]?.toLowerCase();

  if (domain && !FREE_EMAIL_DOMAIN_SET.has(domain)) {
    return domain.toLowerCase();
  }

  return user.name || '';
}

export default function CreateOrganization() {
  const redirectOrNavigate = useRedirectOrNavigate();
  const navigate = useNavigate();
  const { userUpdateLogoutMutation } = useUserApi();
  const { user, isUserLoaded } = useAppContext();

  const logoutMutation = useMutation({
    ...userUpdateLogoutMutation(),
    onSettled: () => {
      queryClient.clear();
      navigate({ to: appRoutes.authLoginRoute.to });
    },
  });

  if (!isUserLoaded) {
    return <></>;
  }

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
            Set up your workspace
          </h2>
          <p className="text-sm text-muted-foreground">
            Create your organization and first workspace to get started with
            Hatchet.
          </p>
        </div>

        <NewOrganizationSaverForm
          defaultOrganizationName={user ? deriveDefaultOrgName(user) : ''}
          defaultTenantName="development"
          afterSave={({ tenant }) => {
            queryClient.prefetchQuery(queries.controlPlane.subscriptionPlans());
            redirectOrNavigate({
              to: appRoutes.tenantOverviewRoute.to,
              params: { tenant: tenant.id },
              replace: true,
            });
          }}
        />
      </AuthLayout>
    </div>
  );
}
