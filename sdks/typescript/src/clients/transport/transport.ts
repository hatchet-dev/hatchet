import type { Interceptor, Transport } from '@connectrpc/connect';

/**
 * The seam every Connect-backed client calls through. A transport carries the protocol
 * (gRPC over HTTP/2 in Node, Connect over fetch elsewhere), the connection settings and the
 * interceptors that apply to every call.
 */
export type { Transport };

/**
 * The characters a token may contain and still travel in an HTTP header: the printable ASCII
 * range, which is also what grpc-js accepts in metadata. Anything else would make the header
 * layer reject the request with an error quoting the token.
 */
const HEADER_SAFE = /^[ -~]*$/;

/**
 * Sets the bearer token the engine authenticates every call with. The transport applies it to
 * each request, so callers never pass credentials per call. A token that cannot be carried in a
 * header is refused here with a fixed message, so the token's value never reaches a log or an
 * error.
 */
export function createAuthInterceptor(token: string): Interceptor {
  if (!HEADER_SAFE.test(token)) {
    throw new Error(
      'The Hatchet client token contains characters that cannot be sent in an HTTP header; check HATCHET_CLIENT_TOKEN for stray whitespace or line breaks'
    );
  }

  return (next) => (req) => {
    req.header.set('authorization', `Bearer ${token}`);
    return next(req);
  };
}
