import { Button, ReviewedButtonTemp } from '@/components/v1/ui/button';
import { PropsWithChildren } from 'react';
import { ErrorResponse, useNavigate, useRouteError } from 'react-router-dom';
import { useLocation } from 'react-router-dom';

export default function ErrorBoundary() {
  const navigate = useNavigate();
  const location = useLocation();

  const error = useRouteError();

  console.error(error);

  const Layout: React.FC<PropsWithChildren> = ({ children }) => (
    <div className="flex flex-row justify-center items-center flex-1 w-full h-full">
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
        <ReviewedButtonTemp onClick={() => window.location.reload()}>
          Reload to Update
        </ReviewedButtonTemp>
        <ReviewedButtonTemp onClick={() => navigate('/')} variant="outline">
          Return to Dashboard
        </ReviewedButtonTemp>
      </Layout>
    );
  }

  if ((error as ErrorResponse).status === 404) {
    return (
      <Layout>
        <h1 className="text-2xl font-semibold tracking-tight">404</h1>
        <h2 className="text-xl font-semibold tracking-tight">Page Not Found</h2>
        <ReviewedButtonTemp onClick={() => navigate('/')}>
          Return to Dashboard
        </ReviewedButtonTemp>
      </Layout>
    );
  }

  return (
    <Layout>
      {(error as ErrorResponse).status && (
        <h1 className="text-2xl font-semibold tracking-tight">
          {(error as ErrorResponse).status}
        </h1>
      )}
      <h2 className="text-xl font-semibold tracking-tight">
        {(error as ErrorResponse).statusText || 'Something went wrong'}
      </h2>

      <ReviewedButtonTemp onClick={() => window.location.reload()}>
        Try Again
      </ReviewedButtonTemp>
      <ReviewedButtonTemp onClick={() => navigate('/')} variant="outline">
        Return to Dashboard
      </ReviewedButtonTemp>
    </Layout>
  );
}
