import HatchetError from '@util/errors/hatchet-error';
import { createChannel } from 'nice-grpc';
import { EventClient } from './event-client';

let client: EventClient;

const mockChannel = createChannel('localhost:50051');

describe('EventClient', () => {
  it('should create a client', () => {
    const x = new EventClient(
      {
        tenant_id: 'TENANT_ID',
        host_port: 'HOST_PORT',
        tls_config: {
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
      },
      mockChannel
    );

    expect(x).toBeDefined();
  });

  beforeEach(() => {
    client = new EventClient(
      {
        tenant_id: 'TENANT_ID',
        host_port: 'HOST_PORT',
        tls_config: {
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
      },
      mockChannel
    );
  });

  fit('should push events', () => {
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
