import { CenterStageLayout } from '@/components/layouts/center-stage.layout';
import { ThemeProvider } from '@/components/theme-provider';
import { ApiConnectionError } from '@/components/errors/api-connection-error';
import useApiMeta from '@/hooks/use-api-meta';
import { PropsWithChildren } from 'react';
import { Outlet } from 'react-router-dom';
import { DocsContext, useDocsState } from '@/hooks/use-docs-sheet';
import { DocsSheetComponent } from '@/components/ui/docs-sheet';

function Root({ children }: PropsWithChildren) {
  const meta = useApiMeta();
  const docsState = useDocsState();

  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <DocsContext.Provider value={docsState}>
        {meta.isLoading ? (
          <CenterStageLayout>Loading...</CenterStageLayout>
        ) : meta.hasFailed ? (
          <ApiConnectionError
            retryInterval={meta.refetchInterval}
            errorMessage={meta.hasFailed.message}
          />
        ) : (
          <>
            {children ?? <Outlet />}
            <DocsSheetComponent
              sheet={docsState.sheet}
              onClose={docsState.close}
            />
          </>
        )}
      </DocsContext.Provider>
    </ThemeProvider>
  );
}

export default Root;
