import { CenterStageLayout } from '@/components/layouts/center-stage.layout';
import { ThemeProvider } from '@/components/theme-provider';
import { ApiConnectionError } from '@/components/errors/api-connection-error';
import useApiMeta from '@/hooks/use-api-meta';
import { PropsWithChildren } from 'react';
import { Outlet } from 'react-router-dom';

function Root({ children }: PropsWithChildren) {
  const meta = useApiMeta();

  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      {meta.isLoading ? (
        <CenterStageLayout>Loading...</CenterStageLayout>
      ) : meta.hasFailed ? (
        <ApiConnectionError
          retryInterval={meta.refetchInterval}
          errorMessage={meta.hasFailed.message}
        />
      ) : (
        <>{children ?? <Outlet />}</>
      )}
    </ThemeProvider>
  );
}

export default Root;
