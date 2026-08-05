import { ServerResponse } from 'node:http';
import { HealthServer, workerStatus } from '@hatchet/v1/client/worker/health-server';

const logger: any = {
  info: jest.fn(),
  warn: jest.fn(),
  debug: jest.fn(),
  error: jest.fn(),
};

function fakeResponse() {
  const chunks: string[] = [];
  const res = {
    writeHead: jest.fn(),
    end: jest.fn((body?: string) => {
      if (body) chunks.push(body);
    }),
  };
  return { res: res as unknown as ServerResponse, body: () => chunks.join('') };
}

function buildServer(slots: number, slotsByPool: Record<string, number>) {
  return new HealthServer(
    8999,
    () => workerStatus.HEALTHY,
    'test-worker',
    () => slots,
    () => slotsByPool,
    () => ['wf:task'],
    () => ({}),
    logger
  );
}

describe('HealthServer slot reporting', () => {
  it('includes the per-pool breakdown alongside the scalar slot count in /health', async () => {
    const server = buildServer(0, { default: 0, durable: 3 });
    const { res, body } = fakeResponse();

    await (server as any).handleHealth(res);

    expect(JSON.parse(body())).toMatchObject({
      status: workerStatus.HEALTHY,
      name: 'test-worker',
      slots: 0,
      slotsByPool: { default: 0, durable: 3 },
    });
  });

  it('exposes a slot_type-labeled gauge on /metrics', async () => {
    const server = buildServer(2, { default: 2, durable: 7 });
    const { res, body } = fakeResponse();

    await (server as any).handleMetrics(res);

    const metrics = body();
    expect(metrics).toContain('hatchet_worker_slots 2');
    expect(metrics).toContain('hatchet_worker_available_slots{slot_type="default"} 2');
    expect(metrics).toContain('hatchet_worker_available_slots{slot_type="durable"} 7');
  });

  it('drops pools that are no longer reported between scrapes', async () => {
    let slotsByPool: Record<string, number> = { default: 2, durable: 7 };
    const server = new HealthServer(
      8999,
      () => workerStatus.HEALTHY,
      'test-worker',
      () => 2,
      () => slotsByPool,
      () => [],
      () => ({}),
      logger
    );

    const first = fakeResponse();
    await (server as any).handleMetrics(first.res);
    expect(first.body()).toContain('hatchet_worker_available_slots{slot_type="durable"} 7');

    slotsByPool = { default: 2 };

    const second = fakeResponse();
    await (server as any).handleMetrics(second.res);
    expect(second.body()).toContain('hatchet_worker_available_slots{slot_type="default"} 2');
    expect(second.body()).not.toContain('slot_type="durable"');
  });
});
