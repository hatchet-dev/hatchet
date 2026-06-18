import { Button } from '@/components/v1/ui/button';
import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';
import { OrganizationSelector } from '@/components/v1/molecules/nav-bar/organization-selector';
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

  const orgName =
    (organizations?.length ?? 0) === 1 ? organizations![0].name : undefined;

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

      <div className="flex h-full items-center justify-center">
        <div className="mx-auto flex max-w-md flex-col items-center gap-6 px-4 text-center">
          <HatchetLogo className="h-10 w-auto" />

          <div className="flex flex-col gap-2">
            <h2 className="text-2xl font-semibold tracking-tight">
              {orgName
                ? `You've joined ${orgName}`
                : "You've joined the organization"}
            </h2>
            <p className="text-sm text-muted-foreground">
              You're not a member of any tenants yet. Contact your organization
              admin to be added to a tenant.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
