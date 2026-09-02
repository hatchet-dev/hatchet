import http from 'node:http';
import { HealthServer, workerStatus, type WorkerStatus } from './health-server';

function noopLogger() {
  return { info: jest.fn(), warn: jest.fn(), error: jest.fn(), debug: jest.fn() };
}

describe('HealthServer /health', () => {
  let server: HealthServer;
  let currentStatus: WorkerStatus = workerStatus.HEALTHY;
  let port = 0;

  beforeEach(async () => {
    currentStatus = workerStatus.HEALTHY;
    server = new HealthServer(
      0,
      () => currentStatus,
      'test-worker',
      () => 10,
      () => ['action-a'],
      () => ({}),
      noopLogger() as any
    );
    await server.start();
    const address = (server as any).server.address();
    port = typeof address === 'object' && address ? address.port : 0;
  });

  afterEach(async () => {
    await server.stop();
  });

  async function getHealthStatus(): Promise<{ statusCode: number; body: any }> {
    return new Promise((resolve, reject) => {
      http
        .get(`http://127.0.0.1:${port}/health`, (res) => {
          let data = '';
          res.on('data', (chunk) => {
            data += chunk;
          });
          res.on('end', () => {
            resolve({ statusCode: res.statusCode ?? 0, body: JSON.parse(data) });
          });
        })
        .on('error', reject);
    });
  }

  it('returns HTTP 200 when worker is HEALTHY', async () => {
    const { statusCode, body } = await getHealthStatus();
    expect(statusCode).toBe(200);
    expect(body.status).toBe(workerStatus.HEALTHY);
  });

  it('returns HTTP 503 when worker is UNHEALTHY', async () => {
    currentStatus = workerStatus.UNHEALTHY;
    const { statusCode, body } = await getHealthStatus();
    expect(statusCode).toBe(503);
    expect(body.status).toBe(workerStatus.UNHEALTHY);
  });
});
