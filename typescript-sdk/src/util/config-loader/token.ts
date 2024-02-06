export function getTenantIdFromJWT(token: string): string {
  const claims = extractClaimsFromJWT(token);
  return claims.sub;
}

export function getAddressesFromJWT(token: string): {
  serverUrl: string;
  grpcBroadcastAddress: string;
} {
  const claims = extractClaimsFromJWT(token);
  return {
    serverUrl: claims.server_url,
    grpcBroadcastAddress: claims.grpc_broadcast_address,
  };
}

function extractClaimsFromJWT(token: string): any {
  const parts = token.split('.');
  if (parts.length !== 3) {
    throw new Error('Invalid token format');
  }

  const claimsPart = parts[1];
  const claimsData = atob(claimsPart.replace(/-/g, '+').replace(/_/g, '/'));
  const claims = JSON.parse(claimsData);

  return claims;
}
