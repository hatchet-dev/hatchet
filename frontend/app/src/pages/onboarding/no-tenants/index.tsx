import { OrganizationSelector } from '@/components/v1/molecules/nav-bar/organization-selector';
import { Button } from '@/components/v1/ui/button';
import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';
import { useUserApi } from '@/lib/api/user-wrapper';
import { useUserUniverse } from '@/providers/user-universe';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { useMutation } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';

export default function NoTenants() {
  const navigate = useNavigate();
  const { userUpdateLogoutMutation } = useUserApi();
  const { organizations, tenantMemberships } = useUserUniverse();

  const logoutMutation = useMutation({
    ...userUpdateLogoutMutation(),
    onSettled: () => {
      queryClient.clear();
      navigate({ to: appRoutes.authLoginRoute.to });
    },
  });

  const primaryOrg = organizations?.[0];
  const orgName =
    (organizations?.length ?? 0) === 1 ? primaryOrg?.name : undefined;

  return (
    <div className="fixed inset-0 z-50 bg-background">
      <div className="absolute top-4 right-4 z-10 flex items-center gap-2">
        <OrganizationSelector memberships={tenantMemberships ?? []} />
        <Button
          variant="ghost"
          size="sm"
          onClick={() => logoutMutation.mutate()}
          disabled={logoutMutation.isPending}
        >
          Sign out
        </Button>
      </div>

      <div className="flex h-full items-center justify-center px-4">
        <div className="flex w-full max-w-lg flex-col gap-6 rounded-lg border bg-card p-6 shadow-sm">
          <div className="flex flex-col gap-3">
            <HatchetLogo variant="mark" className="h-8 w-8" />
            <h2 className="text-2xl font-semibold tracking-tight">
              {orgName
                ? `You've joined ${orgName}`
                : "You've joined the organization"}
            </h2>
            <p className="text-sm text-muted-foreground">
              You're not a member of any tenants yet. Ask an organization owner
              to add you to a tenant, or head to organization settings to manage
              your organization.
            </p>
          </div>

          {primaryOrg && (
            <div className="flex w-full flex-col gap-2">
              <Button
                className="w-full"
                onClick={() =>
                  navigate({
                    to: appRoutes.organizationsIndexRoute.to,
                    params: { organization: primaryOrg.metadata.id },
                  })
                }
              >
                Go to organization settings
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
