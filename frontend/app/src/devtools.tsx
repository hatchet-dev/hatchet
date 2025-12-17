import type { AnyRouter } from '@tanstack/react-router';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { TanStackRouterDevtools } from '@tanstack/router-devtools';

export default function Devtools({ router }: { router: AnyRouter }) {
  let showDevtools = false;
  if (import.meta.env.ROUTER_DEVTOOLS) {
    showDevtools = true;
  }
  return (
    <>
      <ReactQueryDevtools initialIsOpen={false} />
      {showDevtools && (
        <TanStackRouterDevtools position="bottom-right" router={router} />
      )}
    </>
  );
}
