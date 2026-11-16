import { HatchetEmbeddedClient } from '@hatchet-dev/typescript-sdk/v1/embedded';

export async function configuredClient() {
  // > Configure the embedded engine
  const hatchet = await HatchetEmbeddedClient.init({
    // use your own Postgres instead of the bundled one
    databaseUrl: 'postgres://...',
    // store the bundled Postgres runtime and data under this directory
    postgresDataDir: '~/my-project/.hatchet-pg',
    // use RabbitMQ instead of the Postgres message queue
    rabbitmqUrl: 'amqp://...',
    // bind the API / gRPC servers to specific ports
    apiPort: 28243,
    grpcPort: 7070,
    // start only the engine + gRPC, no REST API
    startApi: false,
    // skip running migrations on startup
    runMigrations: false,
    // engine log level (default "warn")
    logLevel: 'info',
    // hatchet-embedded release tag to download
    version: 'v0.105.0',
    // use an existing sidecar binary, skips the download
    binaryPath: '/path/to/hatchet-embedded-sidecar',
    // pinned sha256 of the sidecar binary, replaces checksums.txt as the trust anchor
    checksum: '4f2a...',
  });
  return hatchet;
}

export async function fleetClient() {
  // > Fleet with a shared database
  const hatchet = await HatchetEmbeddedClient.init({
    databaseUrl: 'postgres://user:pass@db.internal:5432/hatchet',
  });
  return hatchet;
}

async function main() {
  // > Create an embedded client
  const hatchet = await HatchetEmbeddedClient.init();

  const greet = hatchet.task({
    name: 'embedded-greet',
    fn: (input: { name: string }) => ({ greeting: `Hello, ${input.name}!` }),
  });

  const worker = await hatchet.worker('embedded-worker', { workflows: [greet] });
  void worker.start();

  const result = await greet.run({ name: 'embed' });
  console.log(result.greeting);

  // > Stop the embedded engine
  await hatchet.stopEmbedded();
  process.exit(0);
}

if (require.main === module) {
  main();
}
