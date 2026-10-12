import { HealthServer, workerStatus, WorkerStatus } from './health-server';

const noopLogger = {
  debug: () => undefined,
  info: () => undefined,
  warn: () => undefined,
  error: () => undefined,
};

async function withServer(
  status: WorkerStatus,
  run: (baseUrl: string) => Promise<void>
): Promise<void> {
  const server = new HealthServer(
    0,
    () => status,
    'test-worker',
    () => 1,
    () => [],
    () => ({}),
    noopLogger as never
  );

  await server.start();
  const address = (
    server as unknown as { server: { address: () => { port: number } } }
  ).server.address();

  try {
    await run(`http://127.0.0.1:${address.port}`);
  } finally {
    await server.stop();
  }
}

describe('HealthServer /health (legacy, unconditional 200)', () => {
  it('still returns 200 for every status, unchanged, for backward compatibility', async () => {
    for (const status of Object.values(workerStatus)) {
      await withServer(status, async (baseUrl) => {
        const response = await fetch(`${baseUrl}/health`);
        const body = (await response.json()) as { status: string };

        expect(response.status).toBe(200);
        expect(body.status).toBe(status);
      });
    }
  });
});

describe('HealthServer /readyz', () => {
  it('returns 200 when the worker is HEALTHY', async () => {
    await withServer(workerStatus.HEALTHY, async (baseUrl) => {
      const response = await fetch(`${baseUrl}/readyz`);

      expect(response.status).toBe(200);
    });
  });

  it('returns 503 when the worker is STARTING', async () => {
    await withServer(workerStatus.STARTING, async (baseUrl) => {
      const response = await fetch(`${baseUrl}/readyz`);

      expect(response.status).toBe(503);
    });
  });

  it('returns 503 when the worker is INITIALIZED', async () => {
    await withServer(workerStatus.INITIALIZED, async (baseUrl) => {
      const response = await fetch(`${baseUrl}/readyz`);

      expect(response.status).toBe(503);
    });
  });

  it('returns 503 when the worker is UNHEALTHY', async () => {
    await withServer(workerStatus.UNHEALTHY, async (baseUrl) => {
      const response = await fetch(`${baseUrl}/readyz`);

      expect(response.status).toBe(503);
    });
  });
});

describe('HealthServer /livez', () => {
  it('returns 200 regardless of worker status, since liveness must not depend on Hatchet connectivity', async () => {
    for (const status of Object.values(workerStatus)) {
      await withServer(status, async (baseUrl) => {
        const response = await fetch(`${baseUrl}/livez`);

        expect(response.status).toBe(200);
      });
    }
  });
});
