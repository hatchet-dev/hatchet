import { PropsWithChildren } from 'react';

export function AuthLayout({ children }: PropsWithChildren) {
  return (
    <div className="flex min-h-full w-full flex-1 flex-col items-center justify-start py-8 lg:flex-row lg:justify-center">
      <div className="container relative w-full flex-col items-center justify-center lg:px-0">
        <div className="mx-auto flex w-full max-w-md lg:p-8">
          <div className="flex w-full flex-col justify-center space-y-6">
            {children}
          </div>
        </div>
      </div>
    </div>
  );
}


