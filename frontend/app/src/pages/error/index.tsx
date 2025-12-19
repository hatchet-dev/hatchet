import { GenericError } from './components/generic-error';
import { NewVersionAvailable } from './components/new-version-available';
import { NotFound } from './components/not-found';
import { TenantForbidden } from './components/tenant-forbidden';
import { appRoutes } from '@/router';
import {
  ErrorComponentProps,
  useMatchRoute,
  useParams,
} from '@tanstack/react-router';
import { isAxiosError } from 'axios';

function getErrorStatus(error: unknown): number | undefined {
  if (!error) {
    return;
  }

  // TanStack Router can throw objects like { status, statusText }
  const maybeStatus = (error as { status?: unknown }).status;
  if (typeof maybeStatus === 'number') {
    return maybeStatus;
  }

  // Axios errors
  if (isAxiosError(error)) {
    const axiosStatus =
      (error as { status?: unknown }).status ?? error.response?.status;
    if (typeof axiosStatus === 'number') {
      return axiosStatus;
    }
  }

  return;
}

export default function ErrorBoundary({ error }: ErrorComponentProps) {
  const matchRoute = useMatchRoute();
  const params = useParams({ strict: false }) as { tenant?: string };
  const status = getErrorStatus(error);

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
      params: params.tenant ? { tenant: params.tenant } : undefined,
      fuzzy: true,
    }),
  );

  if (status === 403 && isTenantRoute) {
    return <TenantForbidden />;
  }

  if ((error as { status?: number }).status === 404) {
    return <NotFound />;
  }

  return (
    <GenericError
      status={status}
      statusText={(error as { statusText?: string }).statusText}
    />
  );
}
