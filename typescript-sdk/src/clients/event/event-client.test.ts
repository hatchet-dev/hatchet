import HatchetError from '@util/errors/hatchet-error';
import { EventClient } from './event-client';
import { mockChannel, mockFactory } from '../hatchet-client/hatchet-client.test';

let client: EventClient;

describe('EventClient', () => {
  it('should create a client', () => {
    const x = new EventClient(
      {
        token: 'TOKEN',
        host_port: 'HOST_PORT',
        tls_config: {
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
      },
      mockChannel,
      mockFactory
    );

    expect(x).toBeDefined();
  });

  beforeEach(() => {
    client = new EventClient(
      {
        token: 'TOKEN',
        host_port: 'HOST_PORT',
        tls_config: {
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
      },
      mockChannel,
      mockFactory
    );
  });

  it('should push events', () => {
    const clientSpy = jest.spyOn(client.client, 'push').mockResolvedValue({
      tenantId: 'x',
      eventId: 'y',
      key: 'z',
      eventTimestamp: new Date(),
      payload: 'string',
    });

    client.push('type', { foo: 'bar' });

    expect(clientSpy).toHaveBeenCalledWith({
      key: 'type',
      payload: '{"foo":"bar"}',
      eventTimestamp: expect.any(Date),
    });
  });

  it('should throw an error when push fails', () => {
    const clientSpy = jest.spyOn(client.client, 'push');
    clientSpy.mockImplementation(() => {
      throw new Error('foo');
    });

    expect(() => {
      client.push('type', { foo: 'bar' });
    }).toThrow(new HatchetError('foo'));
  });
});
