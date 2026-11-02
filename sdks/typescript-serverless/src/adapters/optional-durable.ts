/**
 * Shared by the adapters whose durable relay depends on an optional package (`ws` on Node,
 * `@vercel/functions` and `ws` on Vercel). Whether that package loads is only known
 * asynchronously, and the healthcheck body is built once per handler, so two handlers are
 * built: one advertising the relay and one not. Every request goes to the one the loaded
 * dependency allows, so an endpoint without the package reports `durable.supported: false`
 * and the operator never dials it.
 */
import { createHandler, type HandlerOptions, type ServerlessHandler } from '../handler';
import type { ConsoleLike } from '../handler/context';

export interface OptionalDurableHandler {
  readonly basePath: string;
  matches: ServerlessHandler['matches'];
  /** The handler for this request: durable when the dependency loaded, plain otherwise. */
  select(): Promise<ServerlessHandler>;
}

export function createOptionalDurableHandler(
  options: Omit<HandlerOptions, 'durable'>,
  durableAvailable: Promise<boolean>
): OptionalDurableHandler {
  const out: ConsoleLike = options.console ?? console;
  const durable = createHandler({ ...options, durable: true });
  // The registry warnings were printed by the first handler; the plain one stays quiet.
  const plain = createHandler({
    ...options,
    durable: false,
    console: {
      debug: (...args) => out.debug(...args),
      info: (...args) => out.info(...args),
      warn: () => {},
      error: (...args) => out.error(...args),
    },
  });

  return {
    basePath: durable.basePath,
    matches: (target) => durable.matches(target),
    select: async () => ((await durableAvailable) ? durable : plain),
  };
}
