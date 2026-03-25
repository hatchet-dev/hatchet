import { GenericError } from './components/generic-error';
import { NewVersionAvailable } from './components/new-version-available';
import { NotFound } from './components/not-found';
import { TenantForbidden } from './components/tenant-forbidden';
import { getErrorStatus, getErrorStatusText } from '@/lib/error-utils';
import { getOptionalStringParam } from '@/lib/router-helpers';
import { appRoutes } from '@/router';
import {
  ErrorComponentProps,
  useMatchRoute,
  useParams,
} from '@tanstack/react-router';

export default function ErrorBoundary({ error }: ErrorComponentProps) {
  const matchRoute = useMatchRoute();
  const params = useParams({ strict: false });
  const tenant = getOptionalStringParam(params, 'tenant');
  const status = getErrorStatus(error);
  const statusText = getErrorStatusText(error);

  console.error(error);

  if (
    error instanceof TypeError &&
    error.message.includes('Failed to fetch dynamically imported module:')
  ) {
    return <NewVersionAvailable />;
  }

  const isTenantRoute = Boolean(
    matchRoute({
      to: appRoutes.tenantRoute.to,
      params: tenant ? { tenant } : undefined,
      fuzzy: true,
    }),
  );

  if (status === 403 && isTenantRoute) {
    return <TenantForbidden />;
  }

  if (status === 404) {
    return <NotFound />;
  }

  return <GenericError status={status} statusText={statusText} />;
}
