const { NodeTracerProvider } = require('@opentelemetry/sdk-trace-node');
const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-http');
const { resourceFromAttributes } = require('@opentelemetry/resources');
const { SEMRESATTRS_SERVICE_NAME } = require('@opentelemetry/semantic-conventions');
const { BatchSpanProcessor } = require('@opentelemetry/sdk-trace-base');
const { registerInstrumentations } = require('@opentelemetry/instrumentation');
const { trace } = require('@opentelemetry/api');

import type { TracerProvider, Tracer } from '@opentelemetry/api';
import { HatchetInstrumentor } from '@hatchet-dev/typescript-sdk/opentelemetry';

const isCI = process.env.CI === 'true';

let traceProvider: TracerProvider;

if (isCI) {
  traceProvider = trace.getTracerProvider();
  registerInstrumentations({
    tracerProvider: traceProvider,
    instrumentations: [new HatchetInstrumentor()],
  });
} else {
  const resource = resourceFromAttributes({
    [SEMRESATTRS_SERVICE_NAME]:
      process.env.HATCHET_CLIENT_OTEL_SERVICE_NAME || 'hatchet-typescript-example',
  });

  // Parse headers from environment variable in format "key=value"
  const headersEnv = process.env.HATCHET_CLIENT_OTEL_EXPORTER_OTLP_HEADERS;
  const headers: Record<string, string> | undefined = headersEnv
    ? { [headersEnv.split('=')[0]]: headersEnv.split('=')[1] }
    : undefined;

  const exporter = new OTLPTraceExporter({
    url:
      process.env.HATCHET_CLIENT_OTEL_EXPORTER_OTLP_ENDPOINT || 'http://localhost:4318/v1/traces',
    headers,
  });

  const provider = new NodeTracerProvider({
    resource,
    spanProcessors: [new BatchSpanProcessor(exporter)],
  });

  provider.register();

  traceProvider = provider;


  // NOTE: Instrumentation has to be registered before the instrumented libraries are imported
  registerInstrumentations({
    tracerProvider: traceProvider,
    instrumentations: [
      new HatchetInstrumentor({
        // Optional: exclude sensitive attributes from spans
        // excludedAttributes: ['payload', 'additional_metadata'],

        // Optional: include task name in span names for better filtering
        includeTaskNameInSpanName: true,
      }),
    ],
  });
}


function getTracer(name: string): Tracer {
  return trace.getTracer(name);
}

export { traceProvider, getTracer };
