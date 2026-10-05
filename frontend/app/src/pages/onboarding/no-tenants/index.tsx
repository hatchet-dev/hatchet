import { SetupCard, SetupScreen } from '@/components/layout/setup-card';
import { OrganizationSelector } from '@/components/v1/molecules/nav-bar/organization-selector';
import { Button } from '@/components/v1/ui/button';
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
    <SetupScreen
      topRight={
        <div className="flex items-center gap-2">
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
      }
    >
      <SetupCard
        title={
          orgName
            ? `You've joined ${orgName}`
            : "You've joined the organization"
        }
        footer={
          primaryOrg ? (
            <Button
              size="sm"
              onClick={() =>
                navigate({
                  to: appRoutes.organizationsIndexRoute.to,
                  params: { organization: primaryOrg.metadata.id },
                })
              }
            >
              Go to organization settings
            </Button>
          ) : undefined
        }
      >
        <p className="text-sm text-muted-foreground">
          You're not a member of any tenants yet. Ask an organization owner to
          add you to a tenant, or head to organization settings to manage your
          organization.
        </p>
      </SetupCard>
    </SetupScreen>
  );
}
