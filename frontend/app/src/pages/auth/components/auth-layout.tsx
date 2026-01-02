import { HeroPanel } from './hero-panel';
import { PropsWithChildren } from 'react';

export function AuthLayout({ children }: PropsWithChildren) {
  return (
    <div className="min-h-screen w-full lg:grid lg:grid-cols-2">
      <div className="relative hidden overflow-hidden bg-muted/30 px-10 py-12 lg:flex">
        <div className="pointer-events-none absolute inset-0 bg-gradient-to-br from-primary/10 via-transparent to-transparent" />
        <HeroPanel />
      </div>

      <div className="w-full overflow-y-auto">
        <div className="flex min-h-screen w-full items-center justify-center px-4 py-10 lg:justify-start lg:px-12 lg:py-12">
          <div className="w-full max-w-lg">
            <div className="flex w-full flex-col justify-center space-y-6">
              {children}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
