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

  it('should load from environment variables', () => {
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

  it('should throw an error if the file is not found', () => {
    expect(() =>
      ConfigLoader.load_client_config({
        path: './fixtures/not-found.yaml',
      })
    ).toThrow();
  });

  it('should throw an error if the yaml file fails validation', () => {
    expect(() =>
      ConfigLoader.load_client_config({
        path: './fixtures/.hatchet-invalid.yaml',
      })
    ).toThrow();
  });

  it('should favor yaml config over env vars', () => {
    const config = ConfigLoader.load_client_config({
      path: './fixtures/.hatchet.yaml',
    });
    expect(config).toEqual({
      tenant_id: 'TENANT_ID_YAML',
      host_port: 'HOST_PORT_YAML',
      tls_config: {
        cert_file: 'TLS_CERT_FILE_YAML',
        key_file: 'TLS_KEY_FILE_YAML',
        ca_file: 'TLS_ROOT_CA_FILE_YAML',
        server_name: 'TLS_SERVER_NAME_YAML',
      },
    });
  });

  xit('should attempt to load the root .hatchet.yaml config', () => {
    // TODO i'm not sure the best way to test this, maybe spy on readFileSync called with
    const config = ConfigLoader.load_client_config({
      path: './fixtures/.hatchet.yaml',
    });
    expect(config).toEqual({
      tenant_id: 'TENANT_ID_YAML',
      host_port: 'HOST_PORT_YAML',
      tls_config: {
        cert_file: 'TLS_CERT_FILE_YAML',
        key_file: 'TLS_KEY_FILE_YAML',
        ca_file: 'TLS_ROOT_CA_FILE_YAML',
        server_name: 'TLS_SERVER_NAME_YAML',
      },
    });
  });
});
