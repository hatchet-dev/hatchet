/**
 * Prints the messages a `stream-progress` run published to its durable stream topic, with the
 * Node SDK (`hatchet.streams.events(topic)`, a server stream over gRPC), and exits on the
 * `{"done": true}` message the task publishes last. Reading starts from the oldest retained
 * message, so it works after the run finished as well as while it is publishing.
 *
 *   node scripts/consume-stream.ts <topic>
 *
 * The Node SDK reads HATCHET_CLIENT_TOKEN, which names the engine through its
 * grpc_broadcast_address claim, and for a local engine served over plain HTTP
 * HATCHET_CLIENT_HOST_PORT and HATCHET_CLIENT_TLS_STRATEGY=none, the same variables the
 * endpoint's client reads; `--env-file` loads them from the same file.
 */
import { HatchetClient } from "@hatchet-dev/typescript-sdk";

const topic = process.argv[2];

if (!topic) {
  console.error("usage: node scripts/consume-stream.ts <topic>");
  process.exit(1);
}

const hatchet = HatchetClient.init();

// A topic nothing was published to yet would wait forever; give up after a minute.
const signal = AbortSignal.timeout(60_000);
const decoder = new TextDecoder();

for await (const event of hatchet.streams.events(topic, { signal })) {
  const text = decoder.decode(event.payload);
  console.log(text);

  if (JSON.parse(text).done === true) {
    break;
  }
}

if (signal.aborted) {
  console.error(`no done message on ${topic} within a minute`);
  process.exit(1);
}

process.exit(0);
