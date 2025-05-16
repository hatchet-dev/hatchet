import sleep from '@hatchet-dev/typescript-sdk/util/sleep';
import { randomUUID } from 'crypto';
import { Event } from '@hatchet-dev/typescript-sdk/protoc/events';
import { SIMPLE_EVENT, lower, Input } from './workflow';
import { hatchet } from '../hatchet-client';
import { Worker } from '../../client/worker/worker';

describe('events-e2e', () => {
  let worker: Worker;

  beforeEach(async () => {
    worker = await hatchet.worker('event-worker');
    await worker.registerWorkflow(lower);

    void worker.start();
  });

  afterEach(async () => {
    await worker.stop();
    await sleep(2000);
  })

  async function setupEventFilter(
    testRunId: string,
    expression?: string,
    payload: Record<string, string> = {}
  ) {
    const finalExpression =
      expression || `input.ShouldSkip == false && payload.testRunId == '${testRunId}'`;

    const workflowId = (await hatchet.workflows.get(lower.name)).metadata.id;

    const filter = await hatchet.filters.create({
      workflowId,
      expression: finalExpression,
      scope: testRunId,
      payload: { testRunId, ...payload },
    });

    return async () => {
      await hatchet.filters.delete(filter.metadata.id);
    };
  }

  // Helper function to wait for events to process and fetch runs
  async function waitForEventsToProcess(events: Event[]): Promise<Record<string, any[]>> {
    await sleep(3000);

    const persisted = (await hatchet.events.list({ limit: 100 })).rows || [];

    // Ensure all our events are persisted
    const eventIds = new Set(events.map((e) => e.eventId));
    const persistedIds = new Set(persisted.map((e) => e.metadata.id));
    expect(Array.from(eventIds).every((id) => persistedIds.has(id))).toBeTruthy();

    let attempts = 0;
    const maxAttempts = 15;
    const eventToRuns: Record<string, any[]> = {};

    while (true) {
      console.log('Waiting for event runs to complete...');
      if (attempts > maxAttempts) {
        console.log('Timed out waiting for event runs to complete.');
        return {};
      }

      attempts += 1;

      // For each event, fetch its runs
      const runsPromises = events.map(async (event) => {
        const runs = await hatchet.runs.list({
          triggeringEventExternalId: event.eventId,
        });

        // Extract metadata from event
        const meta = event.additionalMetadata ? JSON.parse(event.additionalMetadata) : {};

        const payload = event.payload ? JSON.parse(event.payload) : {};

        return {
          event: {
            id: event.eventId,
            payload,
            meta,
            shouldHaveRuns: Boolean(meta.should_have_runs),
            testRunId: meta.test_run_id,
          },
          runs: runs.rows || [],
        };
      });

      const eventRuns = await Promise.all(runsPromises);

      // If all events have no runs yet, wait and retry
      if (eventRuns.every(({ runs }) => runs.length === 0)) {
        await sleep(1000);

        // eslint-disable-next-line no-continue
        continue;
      }

      // Store runs by event ID
      for (const { event, runs } of eventRuns) {
        eventToRuns[event.id] = runs;
      }

      // Check if any runs are still in progress
      const anyInProgress = Object.values(eventToRuns).some((runs) =>
        runs.some((run) => run.status === 'QUEUED' || run.status === 'RUNNING')
      );

      if (anyInProgress) {
        await sleep(1000);

        // eslint-disable-next-line no-continue
        continue;
      }

      break;
    }

    return eventToRuns;
  }

  // Helper to verify runs match expectations
  function verifyEventRuns(eventData: any, runs: any[]) {
    if (eventData.shouldHaveRuns) {
      expect(runs.length).toBeGreaterThan(0);
    } else {
      expect(runs.length).toBe(0);
    }
  }

  // Helper to create bulk push event objects
  function createBulkPushEvent({
    index = 1,
    testRunId = '',
    ShouldSkip = false,
    shouldHaveRuns = true,
    key = SIMPLE_EVENT,
    payload = {},
    scope = null,
  }: {
    index?: number;
    testRunId?: string;
    ShouldSkip?: boolean;
    shouldHaveRuns?: boolean;
    key?: string;
    payload?: Record<string, any>;
    scope?: string | null;
  } = {}) {
    return {
      key,
      payload: {
        ShouldSkip,
        Message: `This is event ${index}`,
        ...payload,
      },
      additionalMetadata: {
        should_have_runs: shouldHaveRuns,
        test_run_id: testRunId,
        key: index,
      },
      scope: scope || undefined,
    };
  }

  // Helper to create payload object
  function createEventPayload(ShouldSkip: boolean): Input {
    return { ShouldSkip: ShouldSkip, Message: 'This is event 1' };
  }

  it('should push an event', async () => {
    const event = await hatchet.events.push(SIMPLE_EVENT, createEventPayload(false));
    expect(event.eventId).toBeTruthy();
  }, 10000);

  it('should push an event asynchronously', async () => {
    const event = await hatchet.events.push(SIMPLE_EVENT, createEventPayload(false));
    expect(event.eventId).toBeTruthy();
  }, 10000);

  it('should bulk push events', async () => {
    const events = [
      {
        key: 'event1',
        payload: { Message: 'This is event 1', ShouldSkip: false },
        additionalMetadata: { source: 'test', user_id: 'user123' },
      },
      {
        key: 'event2',
        payload: { Message: 'This is event 2', ShouldSkip: false },
        additionalMetadata: { source: 'test', user_id: 'user456' },
      },
      {
        key: 'event3',
        payload: { Message: 'This is event 3', ShouldSkip: false },
        additionalMetadata: { source: 'test', user_id: 'user789' },
      },
    ];

    const options = { namespace: 'bulk-test' };
    const result = await hatchet.events.bulkPush(SIMPLE_EVENT, events);

    expect(result.events.length).toBe(3);

    // Sort and verify namespacing
    const sortedEvents = [...events].sort((a, b) => a.key.localeCompare(b.key));
    const sortedResults = [...result.events].sort((a, b) => a.key.localeCompare(b.key));
    const namespace = 'bulk-test';

    sortedEvents.forEach((originalEvent, index) => {
      const returnedEvent = sortedResults[index];
      expect(returnedEvent.key).toBe(namespace + originalEvent.key);
    });
  }, 15000);

  it('should process events according to event engine behavior', async () => {
    const testRunId = randomUUID();
    const events = [
      createBulkPushEvent({
        testRunId,
      }),
      createBulkPushEvent({
        testRunId,
        key: 'thisisafakeeventfoobarbaz',
        shouldHaveRuns: false,
      }),
    ];

    console.log('Pushing events:', events);
    const result = await hatchet.events.bulkPush(SIMPLE_EVENT, events);
    console.log('Result:', result);

    const eventToRuns = await waitForEventsToProcess(result.events);

    // Verify each event's runs
    Object.keys(eventToRuns).forEach((eventId) => {
      const runs = eventToRuns[eventId];
      const eventInfo = result.events.find((e) => e.eventId === eventId);

      if (eventInfo) {
        const meta = JSON.parse(eventInfo.additionalMetadata || '{}');
        verifyEventRuns(
          {
            shouldHaveRuns: Boolean(meta.should_have_runs),
          },
          runs
        );
      }
    });
  }, 30000);

  function generateBulkEvents(testRunId: string) {
    return [
      createBulkPushEvent({
        index: 1,
        testRunId,
        ShouldSkip: false,
        shouldHaveRuns: true,
      }),
      createBulkPushEvent({
        index: 2,
        testRunId,
        ShouldSkip: true,
        shouldHaveRuns: true,
      }),
      createBulkPushEvent({
        index: 3,
        testRunId,
        ShouldSkip: false,
        shouldHaveRuns: true,
        scope: testRunId,
      }),
      createBulkPushEvent({
        index: 4,
        testRunId,
        ShouldSkip: true,
        shouldHaveRuns: false,
        scope: testRunId,
      }),
      createBulkPushEvent({
        index: 5,
        testRunId,
        ShouldSkip: true,
        shouldHaveRuns: false,
        scope: testRunId,
        key: 'thisisafakeeventfoobarbaz',
      }),
      createBulkPushEvent({
        index: 6,
        testRunId,
        ShouldSkip: false,
        shouldHaveRuns: false,
        scope: testRunId,
        key: 'thisisafakeeventfoobarbaz',
      }),
    ];
  }

  it('should handle event skipping and filtering', async () => {
    const testRunId = randomUUID();
    const cleanup = await setupEventFilter(testRunId);

    try {
      const events = generateBulkEvents(testRunId);
      const result = await hatchet.events.bulkPush(SIMPLE_EVENT, events);

      const eventToRuns = await waitForEventsToProcess(result.events);

      // Verify each event's runs
      Object.keys(eventToRuns).forEach((eventId) => {
        const runs = eventToRuns[eventId];
        const eventInfo = result.events.find((e) => e.eventId === eventId);

        if (eventInfo) {
          const meta = JSON.parse(eventInfo.additionalMetadata || '{}');
          verifyEventRuns(
            {
              shouldHaveRuns: Boolean(meta.should_have_runs),
            },
            runs
          );
        }
      });
    } finally {
      await cleanup();
    }
  }, 30000);

  async function convertBulkToSingle(event: any) {
    return hatchet.events.push(event.key, event.payload, {
      scope: event.scope,
      additionalMetadata: event.additionalMetadata,
      priority: event.priority,
    });
  }

  it('should handle event skipping and filtering without bulk push', async () => {
    const testRunId = randomUUID();
    const cleanup = await setupEventFilter(testRunId);

    try {
      const rawEvents = generateBulkEvents(testRunId);
      const eventPromises = rawEvents.map((event) => convertBulkToSingle(event));
      const events = await Promise.all(eventPromises);

      const eventToRuns = await waitForEventsToProcess(events);

      // Verify each event's runs
      Object.keys(eventToRuns).forEach((eventId) => {
        const runs = eventToRuns[eventId];
        const eventInfo = events.find((e) => e.eventId === eventId);

        if (eventInfo) {
          const meta = JSON.parse(eventInfo.additionalMetadata || '{}');
          verifyEventRuns(
            {
              shouldHaveRuns: Boolean(meta.should_have_runs),
            },
            runs
          );
        }
      });
    } finally {
      await cleanup();
    }
  }, 30000);

  it('should filter events by payload expression not matching', async () => {
    const testRunId = randomUUID();
    const cleanup = await setupEventFilter(
      testRunId,
      "input.ShouldSkip == false && payload.foobar == 'baz'",
      { foobar: 'qux' }
    );

    try {
      const event = await hatchet.events.push(
        SIMPLE_EVENT,
        { Message: 'This is event 1', ShouldSkip: false },
        {
          scope: testRunId,
          additionalMetadata: {
            should_have_runs: 'false',
            test_run_id: testRunId,
            key: '1',
          },
        }
      );

      const eventToRuns = await waitForEventsToProcess([event]);
      expect(Object.keys(eventToRuns).length).toBe(0);
    } finally {
      await cleanup();
    }
  }, 20000);

  it('should filter events by payload expression matching', async () => {
    const testRunId = randomUUID();
    const cleanup = await setupEventFilter(
      testRunId,
      "input.ShouldSkip == false && payload.foobar == 'baz'",
      { foobar: 'baz' }
    );

    try {
      const event = await hatchet.events.push(
        SIMPLE_EVENT,
        { Message: 'This is event 1', ShouldSkip: false },
        {
          scope: testRunId,
          additionalMetadata: {
            should_have_runs: 'true',
            test_run_id: testRunId,
            key: '1',
          },
        }
      );

      const eventToRuns = await waitForEventsToProcess([event]);
      const runs = Object.values(eventToRuns)[0] || [];
      expect(runs.length).toBeGreaterThan(0);
    } finally {
      await cleanup();
    }
  }, 20000);
});
