import { ChannelCredentials, createChannel, createClientFactory } from 'nice-grpc';
import { InternalHatchetClient } from './hatchet-client';

export const mockChannel = createChannel('localhost:50051');
export const mockFactory = createClientFactory();

describe('Client', () => {
  beforeEach(() => {
    process.env.HATCHET_CLIENT_TOKEN =
      'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef';
  });

  it('should load from environment variables', () => {
    const hatchet = new InternalHatchetClient(
      {
        token:
          'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef',
        host_port: '127.0.0.1:8080',
        log_level: 'OFF',
        namespace: '',
        tls_config: {
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
          tls_strategy: 'tls',
        },
      },
      {
        credentials: ChannelCredentials.createInsecure(),
      }
    );

    expect(hatchet.config).toEqual(
      expect.objectContaining({
        token:
          'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef',
        host_port: '127.0.0.1:8080',
        log_level: 'OFF',
        namespace: '',
        api_url: 'http://localhost:8080',
        tenant_id: '707d0855-80ab-4e1f-a156-f1c4546cbf52',
        tls_config: {
          tls_strategy: 'tls',
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
      })
    );
  });

  it('should throw an error if the config param is invalid', () => {
    expect(
      () =>
        new InternalHatchetClient({
          host_port: 'HOST_PORT',
          tls_config: {
            tls_strategy: 'tls',
            cert_file: 'TLS_CERT_FILE',
            key_file: 'TLS_KEY_FILE',
            ca_file: 'TLS_ROOT_CA_FILE',
            // @ts-ignore
            server_name: undefined,
          },
        })
    ).toThrow();
  });

  it('should favor config param over yaml over env vars ', () => {
    const hatchet = new InternalHatchetClient(
      {
        tls_config: {
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
          tls_strategy: 'tls',
        },
      },
      {
        config_path: './fixtures/.hatchet.yaml',
        credentials: ChannelCredentials.createInsecure(),
      }
    );

    expect(hatchet.config).toEqual(
      expect.objectContaining({
        token:
          'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef',
        host_port: 'HOST_PORT_YAML',
        log_level: 'INFO',
        namespace: '',
        api_url: 'http://localhost:8080',
        tenant_id: '707d0855-80ab-4e1f-a156-f1c4546cbf52',
        tls_config: {
          tls_strategy: 'tls',
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
      })
    );
  });

  describe('Worker', () => {
    let hatchet: InternalHatchetClient;

    beforeEach(() => {
      hatchet = new InternalHatchetClient(
        {
          token:
            'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef',
          host_port: 'HOST_PORT',
          log_level: 'OFF',
          tls_config: {
            cert_file: 'TLS_CERT_FILE',
            key_file: 'TLS_KEY_FILE',
            ca_file: 'TLS_ROOT_CA_FILE',
            server_name: 'TLS_SERVER_NAME',
          },
        },
        {
          credentials: ChannelCredentials.createInsecure(),
        }
      );
    });

    describe('run', () => {
      xit('should start a worker', () => {
        const worker = hatchet.run('workflow1');
        expect(worker).toBeDefined();
      });
    });

    describe('worker', () => {
      it('should start a worker', () => {
        const worker = hatchet.worker('workflow1');

        expect(worker).toBeDefined();
      });
    });
  });
});
