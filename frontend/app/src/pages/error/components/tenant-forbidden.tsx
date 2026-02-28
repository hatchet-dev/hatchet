import { ErrorPageLayout } from './layout';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import { useCurrentUser } from '@/hooks/use-current-user';
import api, { TenantMember } from '@/lib/api';
import { lastTenantAtom } from '@/lib/atoms';
import { getOptionalStringParam } from '@/lib/router-helpers';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import { BuildingOffice2Icon, CheckIcon } from '@heroicons/react/24/outline';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams } from '@tanstack/react-router';
import { useSetAtom } from 'jotai';
import { LogOut, ShieldX, Undo2 } from 'lucide-react';

function TenantPickerItem({
  membership,
  isCurrent,
  onSelect,
}: {
  membership: TenantMember;
  isCurrent: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      disabled={isCurrent}
      className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm hover:bg-muted/50 disabled:opacity-50 disabled:cursor-not-allowed"
    >
      <BuildingOffice2Icon className="size-4 shrink-0" />
      <span className="min-w-0 flex-1 truncate">{membership.tenant?.name}</span>
      {isCurrent && <CheckIcon className="size-4 shrink-0 opacity-50" />}
    </button>
  );
}

export function TenantForbidden() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const tenant = getOptionalStringParam(params, 'tenant');
  const setLastTenant = useSetAtom(lastTenantAtom);
  const queryClient = useQueryClient();

  const { currentUser } = useCurrentUser();
  const { tenantMemberships } = useUserUniverse();

  const logoutMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: async () => {
      await api.userUpdateLogout();
    },
    onSuccess: () => {
      navigate({ to: appRoutes.authLoginRoute.to, replace: true });
    },
  });

  const handleTenantSelect = (membership: TenantMember) => {
    if (!membership.tenant) {
      return;
    }
    setLastTenant(membership.tenant);
    queryClient.clear();
    navigate({
      to: appRoutes.tenantRunsRoute.to,
      params: { tenant: membership.tenant.metadata.id },
    });
  };

  const availableTenants =
    tenantMemberships?.filter((m) => m.tenant?.metadata.id !== tenant) || [];

  return (
    <ErrorPageLayout
      icon={<ShieldX className="h-6 w-6" />}
      title="Access denied"
      description="You don't have permission to view this tenant."
      actions={
        <Button
          leftIcon={<Undo2 className="h-4 w-4" />}
          onClick={() => window.history.back()}
          variant="outline"
        >
          Go back
        </Button>
      }
    >
      <div className="flex justify-center">
        <Badge variant="secondary" className="font-mono">
          403
        </Badge>
      </div>

      <div className="mx-auto w-full max-w-prose rounded-md border bg-muted/20 p-3 text-left font-mono text-xs text-muted-foreground">
        <div className="mb-1 text-[10px] uppercase tracking-wide text-muted-foreground/70">
          Requested Tenant
        </div>
        <div className="break-all text-foreground/90">
          {tenant || 'unknown'}
        </div>
      </div>

      {availableTenants.length > 0 && (
        <div className="mx-auto w-full max-w-prose rounded-md border bg-muted/20 p-2">
          <div className="px-3 pb-1 pt-1 text-[10px] uppercase tracking-wide text-muted-foreground/70">
            Switch to another tenant
          </div>
          <div className="max-h-48 overflow-y-auto">
            {availableTenants.map((membership) => (
              <TenantPickerItem
                key={membership.metadata.id}
                membership={membership}
                isCurrent={false}
                onSelect={() => handleTenantSelect(membership)}
              />
            ))}
          </div>
        </div>
      )}

      <div className="flex flex-row flex-wrap items-center justify-center gap-2">
        {!!currentUser?.email && (
          <div className="text-xs text-muted-foreground">
            Signed in as <span className="font-mono">{currentUser.email}</span>
          </div>
        )}
        <Button
          leftIcon={<LogOut className="size-4" />}
          onClick={() => logoutMutation.mutate()}
          size="sm"
          variant="ghost"
        >
          Sign out
        </Button>
      </div>
    </ErrorPageLayout>
  );
}
