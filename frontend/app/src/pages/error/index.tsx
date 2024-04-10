import { Button } from '@/components/ui/button';
import { ErrorResponse, useNavigate, useRouteError } from 'react-router-dom';

export default function ErrorBoundary() {
  const navigate = useNavigate();
  const error = useRouteError() as ErrorResponse;

  console.error(error);

  return (
    <div className="flex flex-row flex-1 w-full h-full">
      <div className="container relative hidden flex-col items-center justify-center md:grid lg:max-w-none lg:grid-cols-2 lg:px-0">
        <div className="lg:p-8 mx-auto w-screen">
          <div className="mx-auto flex w-40 flex-col justify-center space-y-6 sm:w-[350px]">
            <div className="flex flex-col space-y-2 text-center">
              <h1 className="text-2xl font-semibold tracking-tight">
                {error.statusText || error.status || 'Something went wrong'}
              </h1>
              <Button onClick={() => navigate('/')}>Return to Dashboard</Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
