import { createServer, ServerHttp2Stream } from 'node:http2';
import { AddressInfo } from 'node:net';
import { setTimeout as delay } from 'node:timers/promises';
import { Worker } from 'node:worker_threads';

it('sends a bounded assignment burst while the worker event loop is blocked', async () => {
  const server = createServer();
  let responseStream: ServerHttp2Stream | undefined;
  let advertisedWindow = 0;
  let written = 0;
  const readGate = new Int32Array(new SharedArrayBuffer(4));
  const messageCount = 256;
  const body = Buffer.alloc(656, 'x');
  const frame = Buffer.alloc(body.length + 5);
  frame.writeUInt32BE(body.length, 1);
  body.copy(frame, 5);
  server.on('stream', (stream) => {
    responseStream = stream;
    advertisedWindow = stream.session?.remoteSettings.initialWindowSize ?? 0;
    stream.on('error', () => {});
    stream.resume();
    stream.respond({ ':status': 200, 'content-type': 'application/grpc' });
  });
  await new Promise<void>((resolve) => {
    server.listen(0, '127.0.0.1', resolve);
  });
  const address = server.address() as AddressInfo;
  // The receiver needs its own event loop so blocking it does not also stall the server.
  const receiver = new Worker(
    `
    const { parentPort, workerData } = require('node:worker_threads');
    const { channelFactory } = require(workerData.factory);
    const { ChannelCredentials } = require(workerData.niceGrpc);
    const { Client } = require(workerData.grpc);
    const config = { host_port: workerData.address, tls_config: { tls_strategy: 'none' } };
    const credentials = ChannelCredentials.createInsecure();
    const channel = channelFactory(config, credentials);
    const client = new Client(config.host_port, credentials, { channelOverride: channel });
    const call = client.makeServerStreamRequest('/Dispatcher/ListenV2', () => Buffer.alloc(0), (data) => data, {});
    call.on('error', () => {});
    call.on('metadata', () => {
      parentPort.postMessage('blocked');
      // Keep the receive event loop unavailable until the server has checked its writes.
      Atomics.wait(workerData.readGate, 0, 0, 5000);
      Atomics.store(workerData.readGate, 0, 2);
    });
    let received = 0;
    call.on('data', () => {
      received += 1;
      if (received === workerData.messageCount) parentPort.postMessage('delivered');
    });
  `,
    {
      eval: true,
      execArgv: [
        '-r',
        require.resolve('ts-node/register/transpile-only'),
        '-r',
        require.resolve('tsconfig-paths/register'),
      ],
      workerData: {
        factory: require.resolve('./grpc-helpers'),
        niceGrpc: require.resolve('nice-grpc'),
        grpc: require.resolve('@grpc/grpc-js'),
        address: `127.0.0.1:${address.port}`,
        messageCount,
        readGate,
      },
    }
  );
  const waitForMessage = (message: string) =>
    new Promise<void>((resolve, reject) => {
      receiver.on('message', (value) => {
        if (value === message) resolve();
      });
      receiver.once('error', reject);
    });
  const delivered = waitForMessage('delivered');
  try {
    await waitForMessage('blocked');
    const writesFinished = new Promise<void>((resolve) => {
      const writeNext = () => {
        if (written === messageCount || responseStream?.destroyed) {
          resolve();
          return;
        }
        responseStream?.write(frame, () => {
          written += 1;
          writeNext();
        });
      };
      writeNext();
    });
    await Promise.race([writesFinished, delay(1000)]);
    expect(written).toBe(messageCount);
    expect(advertisedWindow).toBe(4 * 1024 * 1024);
    expect(Atomics.load(readGate, 0)).toBe(0);
    Atomics.store(readGate, 0, 1);
    Atomics.notify(readGate, 0);
    await delivered;
  } finally {
    Atomics.store(readGate, 0, 1);
    Atomics.notify(readGate, 0);
    await receiver.terminate();
    responseStream?.session?.destroy();
    await new Promise<void>((resolve) => {
      server.close(() => resolve());
    });
  }
}, 10000);
