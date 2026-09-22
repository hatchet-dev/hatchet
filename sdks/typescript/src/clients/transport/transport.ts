import type { Interceptor, Transport } from '@connectrpc/connect';

/**
 * The seam every Connect-backed client calls through. A transport carries the protocol
 * (gRPC over HTTP/2 in Node, Connect over fetch elsewhere), the connection settings and the
 * interceptors that apply to every call.
 */
export type { Transport };

/**
 * Sets the bearer token the engine authenticates every call with. The transport applies it to
 * each request, so callers never pass credentials per call.
 */
export function createAuthInterceptor(token: string): Interceptor {
  return (next) => (req) => {
    req.header.set('authorization', `Bearer ${token}`);
    return next(req);
  };
}
