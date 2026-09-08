import type { DurableSocket } from '../handler/durable/socket';

/**
 * Two `DurableSocket`s wired to each other in memory. Frames cross on a microtask, the way
 * a real socket never delivers synchronously, and stay ordered; closing either side closes
 * both and notifies both close listeners once.
 */
export interface SocketPair {
  endpoint: DurableSocket;
  operator: DurableSocket;
  closed: boolean;
}

interface Side extends DurableSocket {
  messageListeners: Array<(text: string) => void>;
  closeListeners: Array<(code: number, reason: string) => void>;
}

export function createSocketPair(): SocketPair {
  const pair = { closed: false } as SocketPair;
  let closeCode = 1005;
  let closeReason = '';

  const side = (peer: () => Side): Side => {
    const self: Side = {
      messageListeners: [],
      closeListeners: [],
      send(text) {
        if (pair.closed) {
          throw new Error('socket is closed');
        }

        // Frames queued before a close are still delivered, in order, ahead of the close.
        queueMicrotask(() => {
          for (const listener of peer().messageListeners) listener(text);
        });
      },
      close(code = 1000, reason = '') {
        if (pair.closed) {
          return;
        }

        pair.closed = true;
        closeCode = code;
        closeReason = reason;

        queueMicrotask(() => {
          for (const listener of [...self.closeListeners, ...peer().closeListeners]) {
            listener(closeCode, closeReason);
          }
        });
      },
      onMessage(listener) {
        self.messageListeners.push(listener);
      },
      onClose(listener) {
        self.closeListeners.push(listener);
      },
    };

    return self;
  };

  // Each side looks its peer up lazily, so the two can reference each other.
  const endpoint: Side = side(() => operator);
  const operator: Side = side(() => endpoint);

  pair.endpoint = endpoint;
  pair.operator = operator;

  return pair;
}
