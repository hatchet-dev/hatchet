import HatchetError from '@util/errors/hatchet-error';
import { DEFAULT_LOGGER } from '@clients/hatchet-client/hatchet-logger';
import { EventClient } from './event-client';
import { mockChannel, mockFactory } from '../hatchet-client/hatchet-client.test';

let client: EventClient;

describe('EventClient', () => {
  it('should create a client', () => {
    const x = new EventClient(
      {
        token:
          'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef',
        host_port: 'HOST_PORT',
        tls_config: {
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
        api_url: 'API_URL',
        tenant_id: 'tenantId',
        logger: DEFAULT_LOGGER,
      },
      mockChannel,
      mockFactory
    );

    expect(x).toBeDefined();
  });

  beforeEach(() => {
    client = new EventClient(
      {
        token:
          'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef',
        host_port: 'HOST_PORT',
        tls_config: {
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
        api_url: 'API_URL',
        tenant_id: 'tenantId',
        logger: DEFAULT_LOGGER,
      },
      mockChannel,
      mockFactory
    );
  });

  it('should push events', async () => {
    const clientSpy = jest.spyOn(client.client, 'push').mockResolvedValue({
      tenantId: 'x',
      eventId: 'y',
      key: 'z',
      eventTimestamp: new Date(),
      payload: 'string',
    });

    await client.push('type', { foo: 'bar' });

    expect(clientSpy).toHaveBeenCalledWith({
      key: 'type',
      payload: '{"foo":"bar"}',
      eventTimestamp: expect.any(Date),
    });
  });

  it('should throw an error when push fails', async () => {
    const clientSpy = jest.spyOn(client.client, 'push');
    clientSpy.mockImplementation(() => {
      throw new Error('foo');
    });

    jest.spyOn(client, 'retrier').mockImplementation((fn, logger, retries, interval) => fn());

    await expect(client.push('type', { foo: 'bar' })).rejects.toThrow(new HatchetError('foo'));
  });

  it('should bulk push events', async () => {
    // Mock the bulkPush method
    const clientSpy = jest.spyOn(client.client, 'bulkPush').mockResolvedValue({
      events: [
        {
          tenantId: 'tenantId',
          eventId: 'y1',
          key: 'z1',
          eventTimestamp: new Date(),
          payload: 'string1',
        },
        {
          tenantId: 'tenantId',
          eventId: 'y2',
          key: 'z2',
          eventTimestamp: new Date(),
          payload: 'string2',
        },
      ],
    });

    // Call bulkPush with an array of events
    const events = [
      { payload: { foo: 'bar1' }, additionalMetadata: { user_id: 'user1' } },
      { payload: { foo: 'bar2' }, additionalMetadata: { user_id: 'user2' } },
    ];
    await client.bulkPush('type', events);

    // Verify the bulkPush method was called with the correct parameters
    expect(clientSpy).toHaveBeenCalledWith({
      events: [
        {
          key: 'type',
          payload: '{"foo":"bar1"}',
          eventTimestamp: expect.any(Date),
          additionalMetadata: '{"user_id":"user1"}',
        },
        {
          key: 'type',
          payload: '{"foo":"bar2"}',
          eventTimestamp: expect.any(Date),
          additionalMetadata: '{"user_id":"user2"}',
        },
      ],
    });
  });

  it('should throw an error when bulkPush fails', async () => {
    // Mock the bulkPush method to throw an error
    const clientSpy = jest.spyOn(client.client, 'bulkPush');
    clientSpy.mockImplementation(() => {
      throw new Error('bulk error');
    });

    jest.spyOn(client, 'retrier').mockImplementation((fn, logger, retries, interval) => fn());

    const events = [
      { payload: { foo: 'bar1' }, additionalMetadata: { user_id: 'user1' } },
      { payload: { foo: 'bar2' }, additionalMetadata: { user_id: 'user2' } },
    ];

    // Test that an error is thrown when bulkPush fails
    await expect(client.bulkPush('type', events)).rejects.toThrow(new HatchetError('bulk error'));
  });
});
