import { Button } from '@/components/v1/ui/button';
import { PropsWithChildren } from 'react';
import {
  ErrorComponentProps,
  useLocation,
  useNavigate,
} from '@tanstack/react-router';
import { appRoutes } from '@/router';

export default function ErrorBoundary({ error }: ErrorComponentProps) {
  const navigate = useNavigate();
  const location = useLocation();

  console.error(error);

  const Layout: React.FC<PropsWithChildren> = ({ children }) => (
    <div className="flex h-full w-full flex-1 flex-row items-center justify-center">
      <div className="flex flex-col space-y-2 text-center">{children}</div>
    </div>
  );

  if (
    error instanceof TypeError &&
    error.message.includes('Failed to fetch dynamically imported module:')
  ) {
    const queryParams = new URLSearchParams(location.search);

    if (!queryParams.has('updated')) {
      queryParams.set('updated', 'true');
      const updatedUrl = `${location.pathname}?${queryParams.toString()}`;
      window.location.href = updatedUrl;
    }

    return (
      <Layout>
        <h1 className="text-2xl font-semibold tracking-tight">
          A New App Version is Available!
        </h1>
        <Button onClick={() => window.location.reload()}>
          Reload to Update
        </Button>
        <Button
          onClick={() => navigate({ to: appRoutes.authenticatedRoute.to })}
          variant="outline"
        >
          Return to Dashboard
        </Button>
      </Layout>
    );
  }

  if ((error as { status?: number }).status === 404) {
    return (
      <Layout>
        <h1 className="text-2xl font-semibold tracking-tight">404</h1>
        <h2 className="text-xl font-semibold tracking-tight">Page Not Found</h2>
        <Button
          onClick={() => navigate({ to: appRoutes.authenticatedRoute.to })}
        >
          Return to Dashboard
        </Button>
      </Layout>
    );
  }

  return (
    <Layout>
      {(error as { status?: number }).status && (
        <h1 className="text-2xl font-semibold tracking-tight">
          {(error as { status?: number }).status}
        </h1>
      )}
      <h2 className="text-xl font-semibold tracking-tight">
        {(error as { statusText?: string }).statusText ||
          'Something went wrong'}
      </h2>

      <Button onClick={() => window.location.reload()}>Try Again</Button>
      <Button
        onClick={() => navigate({ to: appRoutes.authenticatedRoute.to })}
        variant="outline"
      >
        Return to Dashboard
      </Button>
    </Layout>
  );
}
