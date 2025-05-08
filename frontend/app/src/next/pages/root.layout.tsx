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
import { SideSheetContext } from '../hooks/use-side-sheet';
import { useSideSheetState } from '../hooks/use-side-sheet';
import { SideSheetComponent } from '../components/ui/sheet/side-sheet.layout';

function RootContent({ children }: PropsWithChildren) {
  const meta = useApiMeta();
  const docsState = useDocsState();
  const sideSheetState = useSideSheetState();

  return (
    <DocsContext.Provider value={docsState}>
      <SideSheetContext.Provider value={sideSheetState}>
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
          <SideSheetComponent
            sheet={sideSheetState.sheet}
            onClose={sideSheetState.close}
          />
          <DocsSheetComponent
            sheet={docsState.sheet}
            onClose={docsState.close}
          />
        </div>
      )}
    </SideSheetContext.Provider>
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
