import { HealthServer, workerStatus, WorkerStatus } from './health-server';

const noopLogger = {
  debug: () => undefined,
  info: () => undefined,
  warn: () => undefined,
  error: () => undefined,
};

type WorkerState = {
  status: WorkerStatus;
  slots: number;
  actions: string[];
};

async function withServer(
  status: WorkerStatus,
  run: (baseUrl: string, state: WorkerState) => Promise<void>
): Promise<void> {
  const state: WorkerState = { status, slots: 1, actions: [] };
  const server = new HealthServer(
    0,
    () => state.status,
    'test-worker',
    () => state.slots,
    () => state.actions,
    () => ({}),
    noopLogger as never
  );

  await server.start();
  const address = (
    server as unknown as { server: { address: () => { port: number } } }
  ).server.address();

  try {
    await run(`http://127.0.0.1:${address.port}`, state);
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

describe('HealthServer /metrics', () => {
  it('collects default metrics and the current worker values on every scrape', async () => {
    await withServer(workerStatus.HEALTHY, async (baseUrl, state) => {
      const first = await fetch(`${baseUrl}/metrics`);
      const firstMetrics = await first.text();

      expect(first.status).toBe(200);
      expect(first.headers.get('content-type')).toContain('text/plain');
      expect(firstMetrics).toMatch(/^nodejs_version_info\{/m);
      expect(firstMetrics).toMatch(/^hatchet_worker_status 1$/m);
      expect(firstMetrics).toMatch(/^hatchet_worker_slots 1$/m);
      expect(firstMetrics).toMatch(/^hatchet_worker_actions 0$/m);

      state.status = workerStatus.UNHEALTHY;
      state.slots = 3;
      state.actions = ['first', 'second'];

      const second = await fetch(`${baseUrl}/metrics`);
      const secondMetrics = await second.text();

      expect(second.status).toBe(200);
      expect(secondMetrics).toMatch(/^hatchet_worker_status 0$/m);
      expect(secondMetrics).toMatch(/^hatchet_worker_slots 3$/m);
      expect(secondMetrics).toMatch(/^hatchet_worker_actions 2$/m);
    });
  });

  it('serves health checks but reports unavailable metrics when the optional package is missing', async () => {
    jest.doMock('@prometheus-io/client', () => {
      throw new Error('Cannot find module');
    });

    try {
      await withServer(workerStatus.HEALTHY, async (baseUrl) => {
        const health = await fetch(`${baseUrl}/health`);
        const metrics = await fetch(`${baseUrl}/metrics`);

        expect(health.status).toBe(200);
        expect(metrics.status).toBe(503);
        expect(await metrics.text()).toBe('Metrics initialization failed');
      });
    } finally {
      jest.dontMock('@prometheus-io/client');
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
