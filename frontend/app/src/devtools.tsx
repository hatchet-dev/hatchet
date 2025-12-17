import type { AnyRouter } from '@tanstack/react-router';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { TanStackRouterDevtools } from '@tanstack/router-devtools';

export default function Devtools({ router }: { router: AnyRouter }) {
  return (
    <>
      <ReactQueryDevtools initialIsOpen={false} />
      <TanStackRouterDevtools position="bottom-right" router={router} />
    </>
  );
}
