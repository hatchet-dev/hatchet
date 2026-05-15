import sleep from '@hatchet/util/sleep';
import { randomUUID } from 'crypto';
import { Event } from '@hatchet/protoc/events';
import { applyNamespace } from '@hatchet/util/apply-namespace';
import { SIMPLE_EVENT, lower } from './workflow';
import { hatchet } from '../hatchet-client';

describe('events-e2e', () => {
  let testRunId: string;

  beforeEach(() => {
    testRunId = randomUUID();
  });

  async function setupEventFilter(expression?: string, payload: Record<string, string> = {}) {
    const finalExpression =
      expression || `input.should_skip == false && payload.test_run_id == '${testRunId}'`;

    const workflowId = (await hatchet.workflows.get(applyNamespace(lower.name, hatchet.config.namespace))).metadata.id;

    const filter = await hatchet.filters.create({
      workflowId,
      expression: finalExpression,
      scope: testRunId,
      payload: { test_run_id: testRunId, ...payload },
    });

    // Verify the filter is reachable before returning so callers can immediately push events.
    // Without this, events pushed right after filter creation may miss the filter in the engine
    // if it hasn't finished propagating.
    for (let i = 0; i < 50; i += 1) {
      const filters = (await hatchet.filters.list({ scopes: [testRunId] })).rows || [];
      if (filters.some((f) => f.metadata.id === filter.metadata.id)) break;
      await sleep(100);
    }

    return async () => {
      await hatchet.filters.delete(filter.metadata.id);
    };
  }

  // Helper function to wait for events to process and fetch runs
  async function waitForEventsToProcess(events: Event[]): Promise<Record<string, any[]>> {
    const eventIds = events.map((e) => e.eventId);

    // Use eventIds for direct lookup — limit: 100 can miss events in a busy staging environment
    for (let i = 0; i < 150; i += 1) {
      const persisted = (await hatchet.events.list({ eventIds })).rows || [];
      const persistedIds = new Set(persisted.map((e) => e.metadata.id));
      if (eventIds.every((id) => persistedIds.has(id))) break;
      if (i === 149) {
        expect(eventIds.every((id) => persistedIds.has(id))).toBeTruthy();
      }
      await sleep(100);
    }

    const eventToRuns: Record<string, any[]> = {};

    const pollDeadlineMs = Date.now() + 50_000;

    while (Date.now() < pollDeadlineMs) {
      const runsResults = await Promise.all(
        events.map(async (event) => {
          // Query by hatchet__event_id metadata rather than triggeringEventExternalId.
          // Filter-triggered runs (scope-routed events) are not linked via
          // triggeringEventExternalId but do get hatchet__event_id set in their metadata,
          // matching what verifyEventRuns already uses to validate results.
          const runsResp = await hatchet.runs.list({
            additionalMetadata: { hatchet__event_id: event.eventId },
          });
          return { eventId: event.eventId, rawRuns: runsResp.rows || [] };
        })
      );


      if (!runsResults) {
        await sleep(100);
        continue;
      }

      const anyHaveRuns = runsResults.some(({ rawRuns }) => rawRuns.length > 0);

      if (!anyHaveRuns) {
        await sleep(100);
        continue;
      }

      // All runs are terminal (or absent). Store results for all events.
      for (const { eventId, rawRuns } of runsResults) {
        eventToRuns[eventId] = rawRuns;
      }
      console.info(eventToRuns);
      break;
    }

    return eventToRuns;
  }

  // Helper to verify runs match expectations (filter by hatchet__event_id like Python assert_event_runs_processed)
  function verifyEventRuns(eventData: any, runs: any[], eventId: string) {
    const filtered = runs.filter((r) => (r.additionalMetadata || {}).hatchet__event_id === eventId);
    if (eventData.shouldHaveRuns) {
      expect(filtered.length).toBeGreaterThan(0);
    } else {
      expect(filtered.length).toBe(0);
    }
  }

  // Helper to create bulk push event objects (match Python bpi - snake_case in payload)
  function createBulkPushEvent({
    index = 1,
    shouldSkip = false,
    shouldHaveRuns = true,
    key = SIMPLE_EVENT,
    payload = {},
    scope = null,
  }: {
    index?: number;
    shouldSkip?: boolean;
    shouldHaveRuns?: boolean;
    key?: string;
    payload?: Record<string, any>;
    scope?: string | null;
  }) {
    return {
      key,
      payload: {
        should_skip: shouldSkip,
        Message: `This is event ${index}`,
        ...payload,
      },
      additionalMetadata: {
        should_have_runs: shouldHaveRuns,
        test_run_id: testRunId,
        key,
        index,
      },
      scope: scope || undefined,
    };
  }

  // Helper to create payload object (match Python cp - snake_case for filter expression)
  function createEventPayload(shouldSkip: boolean): Record<string, any> {
    return { should_skip: shouldSkip, Message: 'This is event 1' };
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
        key: SIMPLE_EVENT,
        payload: { Message: 'This is event 1', ShouldSkip: false },
        additionalMetadata: { source: 'test', user_id: 'user123' },
      },
      {
        key: SIMPLE_EVENT,
        payload: { Message: 'This is event 2', ShouldSkip: false },
        additionalMetadata: { source: 'test', user_id: 'user456' },
      },
      {
        key: SIMPLE_EVENT,
        payload: { Message: 'This is event 3', ShouldSkip: false },
        additionalMetadata: { source: 'test', user_id: 'user789' },
      },
    ];

    const result = await hatchet.events.bulkPush(SIMPLE_EVENT, events);

    expect(result.events.length).toBe(3);

    // Sort and verify: returned keys are namespaced when client has a namespace
    const sortedEvents = [...events].sort((a, b) => a.key.localeCompare(b.key));
    const sortedResults = [...result.events].sort((a, b) => a.key.localeCompare(b.key));
    const expectedKey = (key: string) => applyNamespace(key, hatchet.config.namespace);

    sortedEvents.forEach((originalEvent, index) => {
      const returnedEvent = sortedResults[index];
      expect(returnedEvent.key).toBe(expectedKey(originalEvent.key));
    });
  }, 15000);

  it('should process events according to event engine behavior', async () => {
    const eventPromises = [
      createBulkPushEvent({ shouldHaveRuns: true }),
      createBulkPushEvent({
        key: 'thisisafakeeventfoobarbaz',
        shouldHaveRuns: false,
      }),
    ].map((event) => convertBulkToSingle(event));
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
          runs,
          eventId
        );
      }
    });
  }, 180_000);

  function generateBulkEvents() {
    return [
      createBulkPushEvent({
        index: 1,
        shouldSkip: false,
        shouldHaveRuns: false,
      }),
      createBulkPushEvent({
        index: 2,
        shouldSkip: true,
        shouldHaveRuns: false,
      }),
      createBulkPushEvent({
        index: 3,
        shouldSkip: false,
        shouldHaveRuns: true,
        scope: testRunId,
      }),
      createBulkPushEvent({
        index: 4,
        shouldSkip: true,
        shouldHaveRuns: false,
        scope: testRunId,
      }),
      createBulkPushEvent({
        index: 5,
        shouldSkip: true,
        shouldHaveRuns: false,
        scope: testRunId,
        key: 'thisisafakeeventfoobarbaz',
      }),
      createBulkPushEvent({
        index: 6,
        shouldSkip: false,
        shouldHaveRuns: false,
        scope: testRunId,
        key: 'thisisafakeeventfoobarbaz',
      }),
    ];
  }

  async function convertBulkToSingle(event: any) {
    return hatchet.events.push(event.key, event.payload, {
      scope: event.scope,
      additionalMetadata: event.additionalMetadata,
      priority: event.priority,
    });
  }

  it('should handle event skipping and filtering without bulk push', async () => {
    const cleanup = await setupEventFilter();

    try {
      const rawEvents = generateBulkEvents();
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
            runs,
            eventId
          );
        }
      });
    } finally {
      await cleanup();
    }
  }, 180_000);

  it('should filter events by payload expression not matching', async () => {
    const cleanup = await setupEventFilter(
      "input.should_skip == false && payload.foobar == 'baz'",
      {
        foobar: 'qux',
      }
    );

    try {
      const event = await hatchet.events.push(
        SIMPLE_EVENT,
        { Message: 'This is event 1', should_skip: false },
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
  }, 120_000);

  it('should filter events by payload expression matching', async () => {
    const cleanup = await setupEventFilter(
      "input.should_skip == false && payload.foobar == 'baz'",
      {
        foobar: 'baz',
      }
    );

    try {
      const event = await hatchet.events.push(
        SIMPLE_EVENT,
        { Message: 'This is event 1', should_skip: false },
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
  }, 90_000);
});
