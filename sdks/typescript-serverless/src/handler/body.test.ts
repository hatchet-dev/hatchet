/**
 * What a POST costs before its signature is checked: a request without a signature reads no
 * body, and a body is read only up to `maxBodyBytes`, whether its size is declared or
 * streamed.
 */
import { describe, expect, it, vi } from 'vitest';
import { SIGNATURE_HEADER, hatchet } from '../index';
import { createTestOperator } from '../testing';

const secret = 'test-secret-at-least-32-characters-long';
const quiet = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() };

// A function of its input, so the test operator POSTs it: the body is what is under test.
const echo = hatchet.task({
  name: 'echo',
  hatchetClient: false,
  fn: async (input: { message: string }) => ({ echo: input.message }),
});

// Shaped like a signature, so the body is read; it verifies against nothing.
const shaped = '0'.repeat(64);

function operator(maxBodyBytes?: number) {
  return createTestOperator({ workflows: [echo], secret, console: quiet, maxBodyBytes });
}

/** A body of `total` bytes in 1 KiB chunks that counts how much of it was pulled. */
function countedBody(total: number) {
  const state = { pulled: 0 };
  const stream = new ReadableStream<Uint8Array>(
    {
      pull(controller) {
        if (state.pulled >= total) {
          controller.close();
          return;
        }

        controller.enqueue(new Uint8Array(1024));
        state.pulled += 1024;
      },
    },
    // No chunk is pulled until something reads.
    { highWaterMark: 0 }
  );

  return { stream, state };
}

function streamed(
  op: ReturnType<typeof operator>,
  route: string,
  body: ReadableStream,
  headers = {}
) {
  const init: RequestInit & { duplex: 'half' } = {
    method: 'POST',
    body,
    headers,
    duplex: 'half',
  };

  return op.handler.fetch(
    new Request(`https://endpoint.test${op.handler.basePath}/${route}`, init)
  );
}

describe('body limits', () => {
  it('reads no body for a request without a signature', async () => {
    const op = operator();

    for (const route of ['trigger', 'healthcheck']) {
      const { stream, state } = countedBody(64 * 1024);
      const response = await streamed(op, route, stream);

      expect(response.status).toBe(401);
      expect(await response.json()).toMatchObject({ error: 'bad signature' });
      expect(state.pulled).toBe(0);
    }
  });

  it('refuses a declared size over maxBodyBytes without reading the body', async () => {
    const op = operator(1024);
    const { stream, state } = countedBody(64 * 1024);
    const response = await streamed(op, 'trigger', stream, {
      [SIGNATURE_HEADER]: shaped,
      'content-length': String(64 * 1024),
    });

    expect(response.status).toBe(413);
    expect(await response.json()).toEqual({
      error: 'the request body exceeds the 1024 byte limit',
      retry: false,
    });
    expect(state.pulled).toBe(0);
  });

  it('stops reading a streamed body at maxBodyBytes', async () => {
    const op = operator(4096);

    for (const route of ['trigger', 'healthcheck']) {
      const { stream, state } = countedBody(1024 * 1024);
      const response = await streamed(op, route, stream, { [SIGNATURE_HEADER]: shaped });

      expect(response.status).toBe(413);
      expect(state.pulled).toBeLessThanOrEqual(8192);
    }
  });

  it('reads a body within the cap and verifies it as before', async () => {
    const op = operator(4096);

    expect(await op.invoke(echo, { message: 'x'.repeat(1024) })).toEqual({
      echo: 'x'.repeat(1024),
    });
    expect(
      (await op.request(`${op.handler.basePath}/trigger`, { body: 'x'.repeat(8192) })).status
    ).toBe(413);
  });
});
