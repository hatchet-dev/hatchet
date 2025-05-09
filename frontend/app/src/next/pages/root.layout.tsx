import { CenterStageLayout } from '@/next/components/layouts/center-stage.layout';
import { ThemeProvider } from '@/next/components/theme-provider';
import { ApiConnectionError } from '@/next/components/errors/api-connection-error';
import useApiMeta from '@/next/hooks/use-api-meta';
import { PropsWithChildren } from 'react';
import { Outlet } from 'react-router-dom';
import { DocsContext, useDocsState } from '@/next/hooks/use-docs-sheet';
import { DocsSheetComponent } from '@/next/components/ui/docs-sheet';
import { TenantProvider } from '../hooks/use-tenant';
import { Toaster } from '@/next/components/ui/toaster';
import { ToastProvider } from '@/next/hooks/utils/use-toast';

function RootContent({ children }: PropsWithChildren) {
  const meta = useApiMeta();
  const docsState = useDocsState();

  return (
    <DocsContext.Provider value={docsState}>
      {meta.isLoading ? (
        <CenterStageLayout>Loading...</CenterStageLayout>
      ) : meta.hasFailed ? (
        <ApiConnectionError
          retryInterval={meta.refetchInterval}
          errorMessage={meta.hasFailed.message}
        />
      ) : (
        <div className="flex h-screen w-full">
          <main className="flex-1 min-w-0 overflow-y-auto overflow-x-hidden">
            {children ?? <Outlet />}
          </main>
          <DocsSheetComponent
            sheet={docsState.sheet}
            onClose={docsState.close}
          />
        </div>
      )}
    </DocsContext.Provider>
  );
}

function Root({ children }: PropsWithChildren) {
  return (
    <ToastProvider>
      <TenantProvider>
        <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
          <Toaster />
          <RootContent>{children}</RootContent>
        </ThemeProvider>
      </TenantProvider>
    </ToastProvider>
  );
}

export default Root;
