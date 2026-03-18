/**
 * Shared OTel instrumentor setup.
 *
 * Import and call initOtel() once at process startup (in both worker and run)
 * before using the tracer.
 */

const { registerInstrumentations } = require('@opentelemetry/instrumentation');
const { trace } = require('@opentelemetry/api');
/* eslint-enable @typescript-eslint/no-require-imports */

import type { Tracer } from '@opentelemetry/api';
import { HatchetInstrumentor } from '@hatchet-dev/typescript-sdk/opentelemetry';

// > Setup
export function initOtel(): void {
  registerInstrumentations({
    instrumentations: [
      new HatchetInstrumentor({
        includeTaskNameInSpanName: true,
      }),
    ],
  });
}

export function getTracer(name: string): Tracer {
  return trace.getTracer(name);
}
