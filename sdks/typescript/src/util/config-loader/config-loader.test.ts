import { ConfigLoader } from './config-loader';

describe('ConfigLoader', () => {
  beforeEach(() => {
    // Clear env vars that might leak from other tests
    delete process.env.HATCHET_CLIENT_TLS_STRATEGY;
    delete process.env.HATCHET_CLIENT_GRPC_MAX_RECV_MESSAGE_LENGTH;
    delete process.env.HATCHET_CLIENT_GRPC_MAX_SEND_MESSAGE_LENGTH;

    process.env.HATCHET_CLIENT_TOKEN =
      'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef';
    process.env.HATCHET_CLIENT_TLS_STRATEGY = 'tls';
    process.env.HATCHET_CLIENT_TLS_CERT_FILE = 'TLS_CERT_FILE';
    process.env.HATCHET_CLIENT_TLS_KEY_FILE = 'TLS_KEY_FILE';
    process.env.HATCHET_CLIENT_TLS_ROOT_CA_FILE = 'TLS_ROOT_CA_FILE';
    process.env.HATCHET_CLIENT_TLS_SERVER_NAME = 'TLS_SERVER_NAME';
    process.env.HATCHET_CLIENT_WORKER_HEALTHCHECK_ENABLED = 'true';
    process.env.HATCHET_CLIENT_WORKER_HEALTHCHECK_PORT = '8001';
    process.env.HATCHET_CLIENT_GRPC_MAX_RECV_MESSAGE_LENGTH = String(8 * 1024 * 1024);
    process.env.HATCHET_CLIENT_GRPC_MAX_SEND_MESSAGE_LENGTH = String(16 * 1024 * 1024);
  });

  it('should load from environment variables', () => {
    const config = ConfigLoader.loadClientConfig();
    expect(config).toEqual({
      host_port: '127.0.0.1:8080',
      log_level: 'INFO',
      namespace: '',
      api_url: 'http://localhost:8080',
      token:
        'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef',
      tenant_id: '707d0855-80ab-4e1f-a156-f1c4546cbf52',
      tls_config: {
        tls_strategy: 'tls',
        cert_file: 'TLS_CERT_FILE',
        key_file: 'TLS_KEY_FILE',
        ca_file: 'TLS_ROOT_CA_FILE',
        server_name: 'TLS_SERVER_NAME',
      },
      healthcheck: {
        enabled: true,
        port: 8001,
      },
      otel: {
        excludedAttributes: [],
        includeTaskNameInSpanName: false,
      },
      grpc_max_recv_message_length: 8 * 1024 * 1024,
      grpc_max_send_message_length: 16 * 1024 * 1024,
    });
  });

  it('should throw on a malformed grpc max recv message length env var', () => {
    process.env.HATCHET_CLIENT_GRPC_MAX_RECV_MESSAGE_LENGTH = '4mb';
    expect(() => ConfigLoader.loadClientConfig()).toThrow(
      /HATCHET_CLIENT_GRPC_MAX_RECV_MESSAGE_LENGTH.*"4mb".*positive integer/
    );
  });

  it('should throw on a malformed grpc max send message length env var', () => {
    process.env.HATCHET_CLIENT_GRPC_MAX_SEND_MESSAGE_LENGTH = '4mb';
    expect(() => ConfigLoader.loadClientConfig()).toThrow(
      /HATCHET_CLIENT_GRPC_MAX_SEND_MESSAGE_LENGTH.*"4mb".*positive integer/
    );
  });

  it('should throw an error if the file is not found', () => {
    expect(() =>
      ConfigLoader.loadClientConfig(
        {},
        {
          path: './fixtures/not-found.yaml',
        }
      )
    ).toThrow();
  });

  xit('should throw an error if the yaml file fails validation', () => {
    expect(() =>
      // This test is failing because there is no invalid state of the yaml file, need to update with tls and mtls settings
      ConfigLoader.loadClientConfig(
        {},
        {
          path: './fixtures/.hatchet-invalid.yaml',
        }
      )
    ).toThrow();
  });

  it('should favor yaml config over env vars', () => {
    const config = ConfigLoader.loadClientConfig(
      {},
      {
        path: './fixtures/.hatchet.yaml',
      }
    );
    expect(config).toEqual({
      token:
        'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef',
      host_port: 'HOST_PORT_YAML',
      log_level: 'INFO',
      namespace: '',
      api_url: 'http://localhost:8080',
      tenant_id: '707d0855-80ab-4e1f-a156-f1c4546cbf52',
      tls_config: {
        tls_strategy: 'tls',
        cert_file: 'TLS_CERT_FILE_YAML',
        key_file: 'TLS_KEY_FILE_YAML',
        ca_file: 'TLS_ROOT_CA_FILE_YAML',
        server_name: 'TLS_SERVER_NAME_YAML',
      },
      healthcheck: {
        enabled: true,
        port: 8002,
      },
      otel: {
        excludedAttributes: ['additional_metadata'],
        includeTaskNameInSpanName: true,
      },
      grpc_max_recv_message_length: 8 * 1024 * 1024,
      grpc_max_send_message_length: 16 * 1024 * 1024,
    });
  });

  xit('should attempt to load the root .hatchet.yaml config', () => {
    //  i'm not sure the best way to test this, maybe spy on readFileSync called with
    const config = ConfigLoader.loadClientConfig(
      {},
      {
        path: './fixtures/.hatchet.yaml',
      }
    );
    expect(config).toEqual({
      token:
        'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef',
      host_port: 'HOST_PORT_YAML',
      tls_config: {
        tls_strategy: 'tls',
        cert_file: 'TLS_CERT_FILE_YAML',
        key_file: 'TLS_KEY_FILE_YAML',
        ca_file: 'TLS_ROOT_CA_FILE_YAML',
        server_name: 'TLS_SERVER_NAME_YAML',
      },
      healthcheck: {
        enabled: true,
        port: 8002,
      },
    });
  });
});
