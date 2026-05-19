/**
 * Shared async context for propagating hatchet.* attributes to child spans.
 *
 * Uses AsyncLocalStorage to store hatchet attributes from the active task run
 * span so that HatchetAttributeSpanProcessor can inject them into all child
 * spans created within the same async context.
 *
 * This mirrors the Go SDK's context.WithValue approach and the Python SDK's
 * ContextVar approach.
 */

import { AsyncLocalStorage } from 'async_hooks';

import type { Attributes } from '@opentelemetry/api';

/**
 * AsyncLocalStorage that holds hatchet.* attributes from the active
 * hatchet task run span so they can be injected into child spans.
 */
export const hatchetSpanAttributes = new AsyncLocalStorage<Attributes>();

/**
 * Store hatchet attributes in async context so the SpanProcessor
 * can inject them into child spans.
 */
export function setHatchetSpanAttributes(attrs: Attributes): void {
  hatchetSpanAttributes.enterWith(attrs);
}
