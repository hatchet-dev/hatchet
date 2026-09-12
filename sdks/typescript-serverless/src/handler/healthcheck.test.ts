import { describe, expect, it, vi } from 'vitest';
import { hatchet } from '../index';
import { createTestOperator } from '../testing';
import { version } from '../../package.json';

const secret = 'test-secret-at-least-32-characters-long';

const echo = hatchet.task({
  name: 'Echo',
  fn: async (input: { message: string }) => ({ echo: input.message }),
});

const pipeline = hatchet.workflow<{ url: string }>({ name: 'pipeline' });
const fetchStep = pipeline.task({ name: 'fetch', fn: async () => ({ body: 'x' }) });
pipeline.task({ name: 'summarize', parents: [fetchStep], fn: async () => ({ words: 1 }) });

const sleeper = hatchet.durableTask({
  name: 'sleep-then-echo',
  fn: async (input: { message: string }) => ({ echo: input.message }),
});

describe('healthcheck', () => {
  it('advertises every workflow as protojson, un-namespaced and lowercased', async () => {
    const op = createTestOperator({ workflows: [echo, pipeline, sleeper], secret });
    const response = await op.request(`${op.handler.basePath}/healthcheck`);
    const raw = JSON.parse(await response.text());

    expect(response.status).toBe(200);
    expect(response.headers.get('content-type')).toBe('application/json');

    // lowerCamelCase field names, as protojson writes them.
    expect(raw.workflows[0]).toMatchObject({
      name: 'echo',
      tasks: [{ readableId: 'Echo', action: 'echo:echo', timeout: '60s' }],
    });
    // protojson leaves default values out.
    expect('isDurable' in raw.workflows[0].tasks[0]).toBe(false);
    expect(raw.workflows[1].tasks.map((t: { readableId: string }) => t.readableId)).toEqual([
      'fetch',
      'summarize',
    ]);
    expect(raw.workflows[1].tasks[1].parents).toEqual(['fetch']);
    expect(raw.workflows[2].tasks[0]).toMatchObject({
      action: 'sleep-then-echo:sleep-then-echo',
      isDurable: true,
    });
    // The test operator dials durable tasks, so support is advertised for the durable one.
    expect(raw.durable).toEqual({ supported: true });
    expect(raw.runtime).toEqual({ name: 'test', sdkVersion: version });
    expect(raw.actions).toEqual([
      'echo:echo',
      'pipeline:fetch',
      'pipeline:summarize',
      'sleep-then-echo:sleep-then-echo',
    ]);
    expect('endpoint_id' in raw).toBe(false);
  });

  it('round-trips through the generated bindings', async () => {
    const op = createTestOperator({ workflows: [echo, pipeline], secret, runtimeName: 'vercel' });
    const decoded = await op.healthcheck();

    expect(decoded.workflows.map((w) => w.name)).toEqual(['echo', 'pipeline']);
    expect(decoded.runtime?.name).toBe('vercel');
    expect(decoded.durable?.supported).toBe(false);
  });

  it('lists only the served subset under actions', async () => {
    const op = createTestOperator({
      workflows: [echo, pipeline],
      serve: [pipeline, 'echo:echo'],
      secret,
    });
    const decoded = await op.healthcheck();

    expect(decoded.actions).toEqual(['echo:echo', 'pipeline:fetch', 'pipeline:summarize']);

    const subset = createTestOperator({ workflows: [echo, pipeline], serve: [echo], secret });

    expect((await subset.healthcheck()).actions).toEqual(['echo:echo']);
    // The definitions are still advertised in full.
    expect((await subset.healthcheck()).workflows).toHaveLength(2);
  });

  it('rejects an unsigned request', async () => {
    const op = createTestOperator({ workflows: [echo], secret });
    const response = await op.request(`${op.handler.basePath}/healthcheck`, {
      body: '{}',
      sign: false,
    });

    expect(response.status).toBe(401);
  });

  it('refuses a serve entry that is not in workflows', () => {
    expect(() => createTestOperator({ workflows: [echo], serve: [pipeline], secret })).toThrow(
      /not in workflows/
    );
  });

  it('warns once per ignored declaration option and leaves it out of the registration', async () => {
    const warn = vi.fn();
    const fancy = hatchet.task({
      name: 'fancy',
      slotCost: 4,
      desiredWorkerLabels: { region: { value: 'eu' } },
      fn: async () => ({}),
    });
    const stickyWorkflow = hatchet.workflow({ name: 'sticky-wf', sticky: 'soft' });
    stickyWorkflow.task({ name: 'a', slotCost: 2, fn: async () => ({}) });
    const evicting = hatchet.durableTask({
      name: 'evicting',
      evictionPolicy: { ttl: '1m' } as never,
      fn: async () => ({}),
    });

    const op = createTestOperator({
      workflows: [fancy, stickyWorkflow, evicting],
      secret,
      console: { debug: vi.fn(), info: vi.fn(), warn, error: vi.fn() },
    });

    const messages = warn.mock.calls.map(([message]) => String(message));

    expect(messages).toHaveLength(4);
    expect(messages).toContainEqual(
      expect.stringMatching(/"slotCost" on fancy:fancy, sticky-wf:a/)
    );
    expect(messages).toContainEqual(expect.stringMatching(/"desiredWorkerLabels" on fancy:fancy/));
    expect(messages).toContainEqual(expect.stringMatching(/"sticky" on sticky-wf/));
    expect(messages).toContainEqual(expect.stringMatching(/"evictionPolicy" on evicting:evicting/));

    const decoded = await op.healthcheck();

    expect(decoded.workflows[0].tasks[0].slotRequests).toEqual({ default: 1 });
    expect(decoded.workflows[0].tasks[0].workerLabels).toEqual({});
    expect(decoded.workflows[1].sticky).toBeUndefined();

    // The declaration itself is untouched.
    expect(fancy.definition._tasks[0].slotCost).toBe(4);
  });
});
