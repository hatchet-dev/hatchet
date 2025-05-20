import { CenterStageLayout } from '@/next/components/layouts/center-stage.layout';
import { ThemeProvider } from '@/next/components/theme-provider';
import { ApiConnectionError } from '@/next/components/errors/api-connection-error';
import useApiMeta from '@/next/hooks/use-api-meta';
import { PropsWithChildren } from 'react';
import { Outlet } from 'react-router-dom';
import { DocsContext, useDocsState } from '@/next/hooks/use-docs-sheet';
import { DocsSheetComponent } from '@/next/components/ui/docs-sheet';
import { Toaster } from '@/next/components/ui/toaster';
import { ToastProvider } from '@/next/hooks/utils/use-toast';
import { SideSheetComponent } from '../components/ui/sheet/side-sheet.layout';
import {
  SideSheetContext,
  useSideSheetState,
} from '@/next/hooks/use-side-sheet';
import { SidebarProvider } from '@/next/components/ui/sidebar';

function RootContent({ children }: PropsWithChildren) {
  const meta = useApiMeta();
  const docsState = useDocsState();
  const sideSheetState = useSideSheetState();

  return (
    <SideSheetContext.Provider value={sideSheetState}>
      <DocsContext.Provider value={docsState}>
        <div className="flex h-screen w-full">
          {meta.isLoading ? (
            <CenterStageLayout>
              <div className="flex h-screen w-full items-center justify-center"></div>
            </CenterStageLayout>
          ) : meta.hasFailed ? (
            <ApiConnectionError
              retryInterval={meta.refetchInterval}
              errorMessage={meta.hasFailed.message}
            />
          ) : (
            <>
              {children ?? <Outlet />}
              <SideSheetComponent />
              <DocsSheetComponent
                sheet={docsState.sheet}
                onClose={docsState.close}
              />
            </>
          )}
        </div>
      </DocsContext.Provider>
    </SideSheetContext.Provider>
  );
}

function Root({ children }: PropsWithChildren) {
  return (
    <ToastProvider>
      <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
        <SidebarProvider>
          <Toaster />
          <RootContent>{children}</RootContent>
        </SidebarProvider>
      </ThemeProvider>
    </ToastProvider>
  );
}

export default Root;
