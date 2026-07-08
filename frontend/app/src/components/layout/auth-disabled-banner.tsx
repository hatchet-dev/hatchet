import { Button } from '@/components/v1/ui/button';
import { useTenantDetails } from '@/hooks/use-tenant';
import { appRoutes } from '@/router';
import {
  ExclamationTriangleIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline';
import { Link } from '@tanstack/react-router';

export function AuthDisabledBanner({ onDismiss }: { onDismiss: () => void }) {
  const { tenantId } = useTenantDetails();

  return (
    <div
      role="alert"
      className="relative flex items-center justify-center gap-3 border-b border-yellow-300 bg-yellow-400 px-10 py-1.5 text-center text-sm font-medium text-yellow-950 dark:border-yellow-700 dark:bg-yellow-500"
    >
      <ExclamationTriangleIcon className="h-4 w-4 flex-shrink-0" />
      You are using an auth-disabled instance of Hatchet.
      {tenantId && (
        <Button
          asChild
          variant="outline"
          size="sm"
          className="h-6 border-yellow-950/30 bg-transparent px-2 text-xs text-yellow-950 hover:bg-yellow-950/10 hover:text-yellow-950"
        >
          <Link
            to={appRoutes.tenantSettingsApiTokensRoute.to}
            params={{ tenant: tenantId }}
          >
            View worker token
          </Link>
        </Button>
      )}
      <Button
        type="button"
        variant="ghost"
        size="icon"
        aria-label="Dismiss"
        onClick={onDismiss}
        className="absolute right-1 top-1/2 h-6 w-6 -translate-y-1/2 text-yellow-950 hover:bg-yellow-950/10 hover:text-yellow-950"
      >
        <XMarkIcon className="h-4 w-4" />
      </Button>
    </div>
  );
}
