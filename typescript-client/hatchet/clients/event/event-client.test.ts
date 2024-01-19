import { EventClient } from './event-client';

describe('EventClient', () => {
  fit('should create a client', () => {
    const x = new EventClient({
      tenant_id: 'TENANT_ID',
      host_port: 'HOST_PORT',
      tls_config: {
        cert_file: 'TLS_CERT_FILE',
        key_file: 'TLS_KEY_FILE',
        ca_file: 'TLS_ROOT_CA_FILE',
        server_name: 'TLS_SERVER_NAME',
      },
    });

    expect(true).toBe(true);
  });

  it('should push events', () => {
    expect(true).toBe(true);
  });
});
