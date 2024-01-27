import { ChannelCredentials, createChannel, createClientFactory } from 'nice-grpc';
import { HatchetClient } from './hatchet-client';

export const mockChannel = createChannel('localhost:50051');
export const mockFactory = createClientFactory();

describe('Client', () => {
  beforeEach(() => {
    process.env.HATCHET_CLIENT_TOKEN = 'TOKEN_ENV';
  });

  it('should load from environment variables', () => {
    const hatchet = new HatchetClient(
      {
        token: 'TOKEN_ENV',
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

    expect(hatchet.config).toEqual({
      token: 'TOKEN_ENV',
      host_port: 'HOST_PORT',
      log_level: 'OFF',
      tls_config: {
        tls_strategy: 'tls',
        cert_file: 'TLS_CERT_FILE',
        key_file: 'TLS_KEY_FILE',
        ca_file: 'TLS_ROOT_CA_FILE',
        server_name: 'TLS_SERVER_NAME',
      },
    });
  });

  it('should throw an error if the config param is invalid', () => {
    expect(
      () =>
        new HatchetClient({
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
    const hatchet = new HatchetClient(
      {
        tls_config: {
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
      },
      {
        config_path: './fixtures/.hatchet.yaml',
        credentials: ChannelCredentials.createInsecure(),
      }
    );

    expect(hatchet.config).toEqual({
      token: 'TOKEN_YAML',
      host_port: 'HOST_PORT_YAML',
      log_level: 'INFO',
      tls_config: {
        tls_strategy: 'tls',
        cert_file: 'TLS_CERT_FILE',
        key_file: 'TLS_KEY_FILE',
        ca_file: 'TLS_ROOT_CA_FILE',
        server_name: 'TLS_SERVER_NAME',
      },
    });
  });

  describe('with_host_port', () => {
    it('should set the host_port', () => {
      const hatchet = HatchetClient.with_host_port(
        'HOST',
        1234,
        {
          token: 'TOKEN',
          tls_config: {
            tls_strategy: 'tls',
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
      expect(hatchet.config).toEqual({
        token: 'TOKEN',
        host_port: 'HOST:1234',
        log_level: 'INFO',
        tls_config: {
          tls_strategy: 'tls',
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
      });
    });
  });

  describe('Worker', () => {
    let hatchet: HatchetClient;

    beforeEach(() => {
      hatchet = new HatchetClient(
        {
          token: 'TOKEN',
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
