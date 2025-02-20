import { getAddressesFromJWT } from './token';

describe('extractClaimsFromJWT', () => {
  it('should correctly extract custom claims from a valid JWT token', () => {
    // Example token, not a real one
    const token =
      'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef';
    const addresses = getAddressesFromJWT(token);
    expect(addresses).toHaveProperty('grpcBroadcastAddress', '127.0.0.1:8080');
    expect(addresses).toHaveProperty('serverUrl', 'http://localhost:8080');
  });

  it('should throw an error for invalid token format', () => {
    const token = 'invalid.token';
    expect(() => getAddressesFromJWT(token)).toThrow('Invalid token format');
  });
});
