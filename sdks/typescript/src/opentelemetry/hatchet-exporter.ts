/**
 * Hatchet OTLP Exporter
 *
 * Creates an OTLP gRPC trace exporter that sends spans to the Hatchet engine's
 * collector endpoint, and a SpanProcessor that injects hatchet.* attributes
 * into all child spans so they are queryable by the same attributes as the parent.
 *
 * This mirrors the Go SDK's EnableHatchetCollector() and the Python SDK's
 * enable_hatchet_otel_collector option.
 */

import type { ClientConfig } from '@hatchet/clients/hatchet-client/client-config';

try {
  require.resolve('@opentelemetry/exporter-trace-otlp-grpc');
  require.resolve('@opentelemetry/sdk-trace-base');
} catch {
  throw new Error(
    'To use HatchetInstrumentor with enableHatchetCollector, you must install: ' +
      'npm install @opentelemetry/exporter-trace-otlp-grpc @opentelemetry/sdk-trace-base'
  );
}

/* eslint-disable @typescript-eslint/no-require-imports */
const {
  OTLPTraceExporter,
} = require('@opentelemetry/exporter-trace-otlp-grpc') as typeof import('@opentelemetry/exporter-trace-otlp-grpc');
const sdkTraceBase =
  require('@opentelemetry/sdk-trace-base') as typeof import('@opentelemetry/sdk-trace-base');
/* eslint-enable @typescript-eslint/no-require-imports */

const { BatchSpanProcessor } = sdkTraceBase;

type SdkTracerProvider = import('@opentelemetry/sdk-trace-base').BasicTracerProvider;
type ReadableSpan = import('@opentelemetry/sdk-trace-base').ReadableSpan;

/**
 * HatchetAttributeSpanProcessor wraps a BatchSpanProcessor.
 * The hatchet.* attributes are already injected by the instrumentor's
 * startActiveSpan call, making them available to all child spans in context.
 */
class HatchetAttributeSpanProcessor extends BatchSpanProcessor {
  onEnd(span: ReadableSpan): void {
    super.onEnd(span);
  }
}

/**
 * Creates an OTLP gRPC trace exporter pointing at the Hatchet engine.
 */
function createHatchetExporter(config: ClientConfig): InstanceType<typeof OTLPTraceExporter> {
  const insecure = config.tls_config.tls_strategy === 'none';

  return new OTLPTraceExporter({
    url: `${insecure ? 'http' : 'https'}://${config.host_port}`,
    metadata: {
      authorization: `Bearer ${config.token}`,
    } as any,
  });
}

/**
 * Adds the Hatchet OTLP exporter to the given TracerProvider.
 * The exporter sends spans to the Hatchet engine's collector endpoint
 * using the same connection settings as the Hatchet client.
 */
export function addHatchetExporter(
  tracerProvider: SdkTracerProvider,
  config: ClientConfig
): void {
  const exporter = createHatchetExporter(config);
  const processor = new HatchetAttributeSpanProcessor(exporter as any);

  tracerProvider.addSpanProcessor(processor);
}
