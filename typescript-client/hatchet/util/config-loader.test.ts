import ConfigLoader from './config-loader';

describe('ConfigLoader', () => {
  beforeEach(() => {
    process.env.HATCHET_CLIENT_TENANT_ID = 'TENANT_ID';
    process.env.HATCHET_CLIENT_HOST_PORT = 'HOST_PORT';
    process.env.HATCHET_CLIENT_TLS_CERT_FILE = 'TLS_CERT_FILE';
    process.env.HATCHET_CLIENT_TLS_KEY_FILE = 'TLS_KEY_FILE';
    process.env.HATCHET_CLIENT_TLS_ROOT_CA_FILE = 'TLS_ROOT_CA_FILE';
    process.env.HATCHET_CLIENT_TLS_SERVER_NAME = 'TLS_SERVER_NAME';
  });

  test('configuration should load from environment variables', () => {
    const config = ConfigLoader.load_client_config();
    expect(config).toEqual({
      tenant_id: 'TENANT_ID',
      host_port: 'HOST_PORT',
      tls_config: {
        cert_file: 'TLS_CERT_FILE',
        key_file: 'TLS_KEY_FILE',
        ca_file: 'TLS_ROOT_CA_FILE',
        server_name: 'TLS_SERVER_NAME',
      },
    });
  });
});
