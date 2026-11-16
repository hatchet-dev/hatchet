import HatchetError from '@util/errors/hatchet-error';
import { Api } from './generated-fetch/Api';

export interface FetchApiOptions {
  /** The API server's base URL, `https://cloud.onhatchet.run` for example; undefined when unknown. */
  apiUrl: string | undefined;
  /** The tenant API token, sent as a bearer token on every request. */
  token: string;
  /** The `fetch` to send with. Defaults to the runtime's global `fetch`. */
  fetch?: typeof globalThis.fetch;
}

/**
 * The generated fetch REST client, authenticated with the token and pointed at the API
 * server. Without a base URL every request rejects with a `HatchetError` that names the
 * config field to set, so a client built from a token that carries no `server_url` claim
 * still serves everything that does not go over REST.
 */
export function createFetchApi(options: FetchApiOptions): Api {
  const send = options.fetch ?? ((input, init) => globalThis.fetch(input, init));
  const noApiUrl = () =>
    Promise.reject(
      new HatchetError(
        'the token carries no server_url claim; set apiUrl on the client config to call the REST API'
      )
    );

  return new Api({
    baseUrl: options.apiUrl ?? '',
    // called as a method of the client, so the runtime's fetch is wrapped rather than passed
    customFetch: options.apiUrl ? (input, init) => send(input, init) : noApiUrl,
    securityWorker: () => ({ headers: { Authorization: `Bearer ${options.token}` } }),
  });
}

/**
 * The REST base URL as the client stores it: an absolute `http://` or `https://` URL with
 * no trailing slash, since the generated client joins it to paths that start with one.
 */
export function normalizeApiUrl(apiUrl: string): string {
  let url: URL;
  try {
    url = new URL(apiUrl);
  } catch {
    throw new HatchetError('apiUrl must be an absolute http:// or https:// URL');
  }
  if (url.protocol !== 'http:' && url.protocol !== 'https:') {
    throw new HatchetError('apiUrl must be an absolute http:// or https:// URL');
  }
  return apiUrl.replace(/\/+$/, '');
}
