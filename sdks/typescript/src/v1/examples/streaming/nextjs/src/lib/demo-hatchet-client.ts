import { HatchetClient } from '@hatchet/v1';

// Demo-specific Hatchet client configuration
// This avoids the need for environment variables in the demo
export const hatchet = HatchetClient.init({
  token: 'demo-token',
  host_port: 'localhost:7077',
  tls_config: {
    tls_strategy: 'none',
  },
  // For demo purposes, we'll catch connection errors gracefully
  namespace: 'demo',
});
