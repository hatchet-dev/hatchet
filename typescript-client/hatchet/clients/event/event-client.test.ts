import HatchetError from '@util/errors/hatchet-error';
import { EventClient } from './event-client';

let client: EventClient;

describe('EventClient', () => {
  it('should create a client', () => {
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

    expect(x).toBeDefined();
  });

  beforeEach(() => {
    client = new EventClient({
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

  it('should push events', () => {
    const clientSpy = jest.spyOn(client.client, 'push');

    client.push('type', { foo: 'bar' });

    expect(clientSpy).toHaveBeenCalledWith({
      tenantId: 'TENANT_ID',
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
