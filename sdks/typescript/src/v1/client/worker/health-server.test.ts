import http from 'node:http';
import { AddressInfo } from 'node:net';
import { HealthServer, workerStatus, WorkerStatus } from './health-server';
import { Logger } from '@hatchet/util/logger';

function getJson(port: number, path: string): Promise<{ statusCode: number; body: any }> {
  return new Promise((resolve, reject) => {
    http
      .get({ host: '127.0.0.1', port, path }, (res) => {
        let data = '';
        res.on('data', (chunk) => {
          data += chunk;
        });
        res.on('end', () => {
          resolve({ statusCode: res.statusCode || 0, body: data ? JSON.parse(data) : undefined });
        });
      })
      .on('error', reject);
  });
}

describe('HealthServer', () => {
  let server: HealthServer;
  let status: WorkerStatus;
  const logger = {
    debug: jest.fn(),
    info: jest.fn(),
    green: jest.fn(),
    warn: jest.fn(),
    error: jest.fn(),
  } as unknown as Logger;

  const startServer = async () => {
    server = new HealthServer(
      0,
      () => status,
      'test-worker',
      () => 5,
      () => ['action-1'],
      () => ({}),
      logger
    );
    await server.start();
    const address = (server as unknown as { server: http.Server }).server.address() as AddressInfo;
    return address.port;
  };

  afterEach(async () => {
    await server.stop();
  });

  it('returns 200 when the worker status is HEALTHY', async () => {
    status = workerStatus.HEALTHY;
    const port = await startServer();

    const { statusCode, body } = await getJson(port, '/health');

    expect(statusCode).toBe(200);
    expect(body.status).toBe(workerStatus.HEALTHY);
  });

  it.each([workerStatus.STARTING, workerStatus.INITIALIZED])(
    'returns 200 when the worker status is %s',
    async (candidateStatus) => {
      status = candidateStatus;
      const port = await startServer();

      const { statusCode, body } = await getJson(port, '/health');

      expect(statusCode).toBe(200);
      expect(body.status).toBe(candidateStatus);
    }
  );

  it('returns 503 when the worker status is UNHEALTHY', async () => {
    status = workerStatus.UNHEALTHY;
    const port = await startServer();

    const { statusCode, body } = await getJson(port, '/health');

    expect(statusCode).toBe(503);
    expect(body.status).toBe(workerStatus.UNHEALTHY);
  });
});
