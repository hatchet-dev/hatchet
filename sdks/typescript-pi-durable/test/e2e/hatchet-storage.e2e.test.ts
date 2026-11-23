// HatchetStorage against a real engine: pi-durable's storage conformance suite on a fresh topic per case, then what
// the suite can't see from one open storage: reopening from the topic, and the topics it must refuse to open.
// Run with PI_E2E=1 and a client configured for the engine (HATCHET_CLIENT_TOKEN and friends).
import { randomUUID } from 'node:crypto';
import type { JsonValue } from '@earendil-works/chord';
import { BACKGROUND_CONTEXT as context } from '@earendil-works/chord/context';
import type { ConversationId, EntryId, StorageWrite, TaskId } from '@earendil-works/pi-durable';
import { commitWrites, registerStorageConformance } from '@earendil-works/pi-durable/testing';
import { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';
import { describe, expect, it } from 'vitest';
import { HatchetStorageError, openHatchetStorage } from '../../src/index.js';

const enabled = process.env.PI_E2E === '1';

const ROOT = 1 as ConversationId;

describe.skipIf(!enabled)('HatchetStorage', () => {
  let client: HatchetClient | undefined;
  const hatchet = () => (client ??= HatchetClient.init());
  const freshTopic = () => `pi-storage-e2e-${randomUUID()}`;

  registerStorageConformance({ describe, expect, it }, 'conformance', async (use) => {
    const storage = await openHatchetStorage(hatchet(), freshTopic());
    try {
      await use(storage);
    } finally {
      await storage.close(context);
    }
  });

  it('reopens with every commit, and mints ids above the stored ones', async () => {
    const topic = freshTopic();
    const first = await openHatchetStorage(hatchet(), topic);
    await commitWrites(first, [{ type: 'conversation', value: { id: ROOT } }], context);
    const entries: EntryId[] = [];
    for (let i = 0; i < 3; i++) {
      const id = await first.mintId<EntryId>();
      entries.push(id);
      await commitWrites(
        first,
        [{ type: 'entry', value: { id, conversationId: ROOT, kind: 'message' } }],
        context
      );
    }
    const taskId = await first.mintId<TaskId<JsonValue>>();
    const task: StorageWrite = {
      type: 'task',
      value: {
        id: taskId,
        conversationId: ROOT,
        kind: 'test.task',
        version: 1,
        input: { step: 1 },
        state: { status: 'pending', checkpoint: { phase: 'ready' } },
        background: false,
        abortRequested: false,
      },
    };
    // Two batches in one call go out as one message.
    const seqs: number[] = [];
    const result = await first.commit(
      [
        [task],
        [
          {
            type: 'entry',
            value: { id: await first.mintId<EntryId>(), conversationId: ROOT, kind: 'message' },
          },
        ],
      ],
      (_index, seq) => seqs.push(seq),
      context
    );
    expect(result.stored).toBe(2);
    const before = await first.scanEntries({ conversationId: ROOT }, 100, undefined, context);
    await first.close(context);

    const second = await openHatchetStorage(hatchet(), topic);
    try {
      expect(await second.conversation(ROOT, context)).toEqual({ id: ROOT });
      expect(await second.scanEntries({ conversationId: ROOT }, 100, undefined, context)).toEqual(
        before
      );
      expect((await second.entry(entries[1], context))?.commitSeq).toBeDefined();
      expect(await second.task(taskId, context)).toMatchObject({
        id: taskId,
        state: { status: 'pending' },
      });
      const stored = [...entries, taskId, ...before.items.map((entry) => entry.id)];
      expect(await second.mintId()).toBeGreaterThan(Math.max(...stored));

      // The reopened storage keeps appending after the commits it read.
      await commitWrites(
        second,
        [
          {
            type: 'entry',
            value: { id: await second.mintId<EntryId>(), conversationId: ROOT, kind: 'message' },
          },
        ],
        context
      );
    } finally {
      await second.close(context);
    }

    const third = await openHatchetStorage(hatchet(), topic);
    try {
      const page = await third.scanEntries({ conversationId: ROOT }, 100, undefined, context);
      expect(page.items).toHaveLength(before.items.length + 1);
    } finally {
      await third.close(context);
    }
  });

  it('refuses a topic whose first message is not its header', async () => {
    // What a topic looks like once retention has dropped its oldest messages.
    const topic = freshTopic();
    await hatchet().streams.publish(
      topic,
      JSON.stringify({
        format: 1,
        type: 'commit',
        commits: [{ seq: 5, writes: [{ type: 'conversation', value: { id: ROOT } }] }],
      })
    );
    await expect(openHatchetStorage(hatchet(), topic)).rejects.toThrow(HatchetStorageError);
    await expect(openHatchetStorage(hatchet(), topic)).rejects.toThrow(/stream retention/);
  });

  it('refuses a topic two owners wrote', async () => {
    const topic = freshTopic();
    const owner = await openHatchetStorage(hatchet(), topic);
    await commitWrites(owner, [{ type: 'conversation', value: { id: ROOT } }], context);
    // A second process opens the same storage while the first still runs, and both commit seq 2.
    const intruder = await openHatchetStorage(hatchet(), topic);
    for (const storage of [owner, intruder]) {
      await commitWrites(
        storage,
        [
          {
            type: 'entry',
            value: { id: await storage.mintId<EntryId>(), conversationId: ROOT, kind: 'message' },
          },
        ],
        context
      );
      await storage.close(context);
    }
    await expect(openHatchetStorage(hatchet(), topic)).rejects.toThrow(/two owners/);
  });
});
