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

import { hatchetSpanAttributes } from './hatchet-span-context';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let otelDiag: any;
try {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  otelDiag = (require('@opentelemetry/api') as typeof import('@opentelemetry/api')).diag;
} catch {
  // best-effort
}

try {
  require.resolve('@opentelemetry/exporter-trace-otlp-grpc');
  require.resolve('@opentelemetry/sdk-trace-base');
  require.resolve('@opentelemetry/core');
} catch {
  throw new Error(
    'To use HatchetInstrumentor with enableHatchetCollector, you must install: ' +
      'npm install @opentelemetry/exporter-trace-otlp-grpc @opentelemetry/sdk-trace-base @opentelemetry/core'
  );
}

/* eslint-disable @typescript-eslint/no-require-imports */
// prettier-ignore
const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-grpc') as typeof import('@opentelemetry/exporter-trace-otlp-grpc');
// prettier-ignore
const sdkTraceBase = require('@opentelemetry/sdk-trace-base') as typeof import('@opentelemetry/sdk-trace-base');
// prettier-ignore
const { ExportResultCode } = require('@opentelemetry/core') as typeof import('@opentelemetry/core');
/* eslint-enable @typescript-eslint/no-require-imports */

const { BatchSpanProcessor } = sdkTraceBase;

type ReadableSpan = import('@opentelemetry/sdk-trace-base').ReadableSpan;
type SdkSpan = import('@opentelemetry/sdk-trace-base').Span;
type SpanExporter = import('@opentelemetry/sdk-trace-base').SpanExporter;
type ExportResult = import('@opentelemetry/core').ExportResult;

const GRPC_STATUS_UNIMPLEMENTED = 12;
const RETRY_AFTER_MS = 5 * 60 * 1000;

class HatchetExporterWrapper implements SpanExporter {
  private inner: SpanExporter;
  private retryAt = 0;

  constructor(inner: SpanExporter) {
    this.inner = inner;
  }

  export(spans: ReadableSpan[], resultCallback: (result: ExportResult) => void): void {
    if (this.retryAt > 0 && Date.now() < this.retryAt) {
      resultCallback({ code: ExportResultCode.SUCCESS });
      return;
    }

    try {
      this.inner.export(spans, (result: ExportResult) => {
        if (result.code !== ExportResultCode.SUCCESS && result.error) {
          const err = result.error as unknown as Record<string, unknown>;
          if (
            err.code === GRPC_STATUS_UNIMPLEMENTED ||
            err.message?.toString().includes('UNIMPLEMENTED')
          ) {
            this.retryAt = Date.now() + RETRY_AFTER_MS;
            resultCallback({ code: ExportResultCode.SUCCESS });
            return;
          }
        }
        this.retryAt = 0;
        resultCallback(result);
      });
    } catch (e: unknown) {
      if (e instanceof TypeError && e.message?.includes("reading 'name'")) {
        otelDiag?.error(
          'hatchet instrumentation: OpenTelemetry package version mismatch. ' +
            '@opentelemetry/exporter-trace-otlp-grpc and @opentelemetry/sdk-trace-base must be ' +
            'from the same release set (1.x + 0.5x.x, or 2.x + 0.20x.x). ' +
            'See https://github.com/open-telemetry/opentelemetry-js#version-compatibility'
        );
        resultCallback({ code: ExportResultCode.SUCCESS });
        return;
      }
      throw e;
    }
  }

  shutdown(): Promise<void> {
    return this.inner.shutdown();
  }

  forceFlush(): Promise<void> {
    return this.inner.forceFlush?.() ?? Promise.resolve();
  }
}

/**
 * HatchetAttributeSpanProcessor wraps a BatchSpanProcessor and injects
 * hatchet.* attributes into every span created within a step run context.
 * This ensures child spans are queryable by the same attributes (e.g.
 * hatchet.step_run_id) as the parent span.
 */
class HatchetAttributeSpanProcessor extends BatchSpanProcessor {
  onStart(span: SdkSpan): void {
    const attrs = hatchetSpanAttributes.getStore();
    if (attrs) {
      span.setAttributes(attrs);
    }
    super.onStart(span, undefined as never);
  }

  onEnd(span: ReadableSpan): void {
    super.onEnd(span);
  }
}

function createHatchetExporter(config: ClientConfig): InstanceType<typeof OTLPTraceExporter> {
  const insecure = config.tls_config.tls_strategy === 'none';

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const opts: Record<string, any> = {
    url: `${insecure ? 'http' : 'https'}://${config.host_port}`,
    metadata: { authorization: `Bearer ${config.token}` },
  };

  if (!insecure && config.tls_config.ca_file) {
    try {
      /* eslint-disable @typescript-eslint/no-require-imports */
      const fs = require('fs') as typeof import('fs');
      const { ChannelCredentials } = require('nice-grpc') as typeof import('nice-grpc');
      /* eslint-enable @typescript-eslint/no-require-imports */
      const rootCerts = fs.readFileSync(config.tls_config.ca_file);
      opts.credentials = ChannelCredentials.createSsl(rootCerts);
    } catch {
      // Fall through to default TLS handling
    }
  }

  return new OTLPTraceExporter(opts);
}

export interface HatchetBspConfig {
  scheduledDelayMillis?: number;
  maxExportBatchSize?: number;
  maxQueueSize?: number;
}

/**
 * Creates a SpanProcessor that sends spans to the Hatchet engine's
 * collector endpoint using the same connection settings as the Hatchet client.
 * Pass the returned processor to BasicTracerProvider's `spanProcessors` option.
 */
export function createHatchetSpanProcessor(
  config: ClientConfig,
  bspConfig?: HatchetBspConfig
): InstanceType<typeof HatchetAttributeSpanProcessor> {
  const inner = createHatchetExporter(config);
  const exporter = new HatchetExporterWrapper(inner as unknown as SpanExporter);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return new HatchetAttributeSpanProcessor(exporter as any, {
    scheduledDelayMillis: bspConfig?.scheduledDelayMillis,
    maxExportBatchSize: bspConfig?.maxExportBatchSize,
    maxQueueSize: bspConfig?.maxQueueSize,
  });
}
