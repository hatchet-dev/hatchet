/**
 * Prints the messages a `stream-progress` run published to its durable stream topic, with the
 * Node SDK (`hatchet.streams.events(topic)`, a server stream over gRPC), and exits on the
 * `{"done": true}` message the task publishes last. Reading starts from the oldest retained
 * message, so it works after the run finished as well as while it is publishing.
 *
 *   node scripts/consume-stream.ts <topic>
 *
 * HATCHET_CLIENT_TOKEN names the engine through its grpc_broadcast_address claim. For a local
 * engine served over plain HTTP, HATCHET_CLIENT_SERVER_URL (http://localhost:7070, as the
 * endpoint's client uses it) sets the address and turns TLS off; `--env-file` loads it from
 * the same file the endpoint reads.
 */
import { HatchetClient } from "@hatchet-dev/typescript-sdk";

const topic = process.argv[2];

if (!topic) {
  console.error("usage: node scripts/consume-stream.ts <topic>");
  process.exit(1);
}

const serverUrl = process.env.HATCHET_CLIENT_SERVER_URL;
const local = serverUrl
  ? {
      host_port: new URL(serverUrl).host,
      tls_config: { tls_strategy: serverUrl.startsWith("http:") ? ("none" as const) : ("tls" as const) },
    }
  : {};

const hatchet = HatchetClient.init({ token: process.env.HATCHET_CLIENT_TOKEN, ...local });

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
