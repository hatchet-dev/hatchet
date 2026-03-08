import { parseJSON } from '../parse';

export function getTenantIdFromJWT(token: string): string {
  const claims = extractClaimsFromJWT(token);
  if (claims.sub === undefined) {
    throw new Error('Invalid token: missing sub');
  }
  return claims.sub;
}

export function getAddressesFromJWT(token: string): {
  serverUrl: string;
  grpcBroadcastAddress: string;
} {
  const claims = extractClaimsFromJWT(token);
  if (claims.server_url === undefined || claims.grpc_broadcast_address === undefined) {
    throw new Error('Invalid token: missing server_url or grpc_broadcast_address');
  }
  return {
    serverUrl: claims.server_url,
    grpcBroadcastAddress: claims.grpc_broadcast_address,
  };
}

interface JWTClaims {
  sub?: string;
  server_url?: string;
  grpc_broadcast_address?: string;
  [key: string]: unknown;
}

function extractClaimsFromJWT(token: string): JWTClaims {
  const parts = token.split('.');
  if (parts.length !== 3) {
    throw new Error('Invalid token format');
  }

  const [_, claimsPart] = parts;
  const claimsData = atob(claimsPart.replace(/-/g, '+').replace(/_/g, '/'));
  const claims = parseJSON<JWTClaims>(claimsData);

  return claims;
}
