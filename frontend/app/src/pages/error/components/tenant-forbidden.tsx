import { ErrorPageLayout } from './layout';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import api, { queries } from '@/lib/api';
import { getOptionalStringParam } from '@/lib/router-helpers';
import { appRoutes } from '@/router';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate, useParams } from '@tanstack/react-router';
import { LogOut, ShieldX, Undo2 } from 'lucide-react';

export function TenantForbidden() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const tenant = getOptionalStringParam(params, 'tenant');

  const userQuery = useQuery({
    ...queries.user.current,
    retry: false,
  });

  const logoutMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: async () => {
      await api.userUpdateLogout();
    },
    onSuccess: () => {
      navigate({ to: appRoutes.authLoginRoute.to, replace: true });
    },
  });

  return (
    <ErrorPageLayout
      icon={<ShieldX className="h-6 w-6" />}
      title="Access denied"
      description="You don’t have permission to view this tenant."
      actions={
        <>
          <Button
            leftIcon={<Undo2 className="h-4 w-4" />}
            onClick={() => window.history.back()}
            variant="outline"
          >
            Go back
          </Button>
          <Button
            onClick={() =>
              navigate({ to: appRoutes.authenticatedRoute.to, replace: true })
            }
          >
            Switch tenant
          </Button>
        </>
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

      <div className="flex flex-row flex-wrap items-center justify-center gap-2">
        {!!userQuery.data?.email && (
          <div className="text-xs text-muted-foreground">
            Signed in as{' '}
            <span className="font-mono">{userQuery.data.email}</span>
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
