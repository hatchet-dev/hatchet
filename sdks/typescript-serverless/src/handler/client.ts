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
 * A function called with an environment that is not an object (a string tenant key, say)
 * gets a client per request, since two such values cannot be told apart by identity and must
 * never share a credential.
 */
import { HatchetCore, type CoreClientConfig } from '@hatchet-dev/typescript-sdk/core/index.js';

export type { CoreClientConfig, HatchetCore };

/**
 * The variables the adapters build the default client from, named as the Node SDK names them so
 * one `.env` file serves a worker, a script and an endpoint alike. Without a token there is no
 * client.
 */
export interface ClientEnv {
  /** A tenant API token; its claims name the engine and the API server. */
  HATCHET_CLIENT_TOKEN?: string;
  /** The engine's `host:port` when it is not the one the token names: a local engine under a dev server. */
  HATCHET_CLIENT_HOST_PORT?: string;
  /** `none` for an engine served over plain HTTP; any other value keeps TLS. */
  HATCHET_CLIENT_TLS_STRATEGY?: string;
}

/** The default client config of an adapter: `undefined` without a token, else the token with the host and TLS overrides that are set. */
export function clientConfigFromEnv(
  env: ClientEnv | Readonly<Record<string, string | undefined>> | undefined
): CoreClientConfig | undefined {
  const token = env?.HATCHET_CLIENT_TOKEN;

  if (!token) {
    return undefined;
  }

  const config: CoreClientConfig = { token };

  if (env?.HATCHET_CLIENT_HOST_PORT) {
    const hostPort = env.HATCHET_CLIENT_HOST_PORT;

    // The client turns it into the base URL of its calls, so a path, query, fragment or user
    // info in it would become a prefix of every call instead of failing; a URL belongs in
    // the `serverUrl` option, not here. The resolver forms the client accepts (`dns:`,
    // `dns:///`, `ipv4:`, `ipv6:`) are allowed in front of the host and port.
    const bare = hostPort.replace(/^(?:dns|ipv4|ipv6):(?:\/\/[^/]*\/)?/, '');

    if (/[/?#@\s]/.test(bare)) {
      throw new Error(
        `HATCHET_CLIENT_HOST_PORT must be a host and port such as localhost:7070, got ${JSON.stringify(hostPort)}; a full URL goes in the client's serverUrl option`
      );
    }

    config.hostPort = hostPort;
  }

  if (env?.HATCHET_CLIENT_TLS_STRATEGY === 'none') {
    config.tls = { strategy: 'none' };
  }

  return config;
}

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
 * object when the option is a function, for the life of the handler; a function option
 * called with a non-object environment gets a client per request.
 */
export function createClientResolver(
  option: ClientOption | undefined
): (env: unknown) => ClientSource {
  const clients = new WeakMap<object, HatchetCore>();

  const build = (key: object | undefined, config: CoreClientConfig): HatchetCore => {
    if (key === undefined) {
      return new HatchetCore(config);
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
