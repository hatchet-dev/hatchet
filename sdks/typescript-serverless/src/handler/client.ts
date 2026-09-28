/**
 * The optional Hatchet client of a handler. It is the SDK's core client (`HatchetCore` from
 * `@hatchet-dev/typescript-sdk/core`): unary Connect calls over `fetch`, so it runs where the
 * handler runs. The token comes from the platform's secret store, the same way the signing
 * secret does; the operator never sends it.
 *
 * The option takes a ready client, the config `HatchetCore` takes, or a function of the
 * runtime's environment returning either, since some runtimes (Cloudflare Workers) only hand
 * secrets to the request. A config is turned into one client per handler, or per environment
 * object when it comes from a function, and that client is reused; it is built the first time
 * a task needs it, so a task that never touches the client never pays for it or fails on it.
 */
import { HatchetCore, type CoreClientConfig } from '@hatchet-dev/typescript-sdk/core/index.js';

export type { CoreClientConfig, HatchetCore };

/** What the `client` option accepts. */
export type ClientOption =
  HatchetCore | CoreClientConfig | ((env: unknown) => HatchetCore | CoreClientConfig | undefined);

/** Hands a task the handler's client, building it on first use; `undefined` without one. */
export type ClientSource = () => HatchetCore | undefined;

/** Whether the value is a client instance rather than a config; duck-typed, so a client from another copy of the SDK counts. */
export function isHatchetCore(value: HatchetCore | CoreClientConfig): value is HatchetCore {
  const candidate = value as Partial<HatchetCore>;

  return typeof candidate.runNoWait === 'function' && typeof candidate.runs === 'object';
}

/**
 * Builds the per-request client source for a handler: `resolve(env)` returns the function a
 * task calls to get the client. One client is kept per config object, or per environment
 * object when the option is a function, for the life of the handler.
 */
export function createClientResolver(
  option: ClientOption | undefined
): (env: unknown) => ClientSource {
  const clients = new WeakMap<object, HatchetCore>();
  // The one client of a function option called with an environment that is not an object.
  let unkeyed: HatchetCore | undefined;

  const build = (key: object | undefined, config: CoreClientConfig): HatchetCore => {
    if (key === undefined) {
      unkeyed ??= new HatchetCore(config);
      return unkeyed;
    }

    let client = clients.get(key);

    if (!client) {
      client = new HatchetCore(config);
      clients.set(key, client);
    }

    return client;
  };

  return (env) => {
    let source: ClientSource | undefined;

    return () => {
      if (source) {
        return source();
      }

      const resolved = typeof option === 'function' ? option(env) : option;

      if (!resolved) {
        source = () => undefined;
        return undefined;
      }

      if (isHatchetCore(resolved)) {
        source = () => resolved;
        return resolved;
      }

      const key =
        typeof option === 'function'
          ? typeof env === 'object' && env !== null
            ? env
            : undefined
          : resolved;
      const client = build(key, resolved);
      source = () => client;

      return client;
    };
  };
}
