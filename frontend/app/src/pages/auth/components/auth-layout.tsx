import { PropsWithChildren } from 'react';

export function AuthLayout({ children }: PropsWithChildren) {
  return (
    <div className="flex min-h-screen w-full">
      {/* Left column: advertisement surface (hidden on mobile) */}
      <div className="hidden w-1/2 flex-col justify-center bg-muted/40 p-10 lg:flex">
        <div className="mx-auto w-full max-w-lg">
          <div className="text-sm font-medium text-muted-foreground">
            advertisment surface
          </div>
        </div>
      </div>

      {/* Right column: form (responsible for its own scroll) */}
      <div className="w-full overflow-y-auto lg:w-1/2">
        <div className="flex min-h-screen w-full items-center justify-center px-4 py-10 lg:justify-start lg:pl-12 lg:pr-10 lg:py-12">
          <div className="w-full max-w-lg">
            <div className="flex w-full flex-col justify-center space-y-7">
              {children}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}


