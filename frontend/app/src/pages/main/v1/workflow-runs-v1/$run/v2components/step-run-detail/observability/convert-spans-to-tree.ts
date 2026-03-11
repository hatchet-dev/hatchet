import type {
  OpenTelemetrySpan,
  TraceSpan,
  TraceSpanAttribute,
} from './agent-prism-types';
import { INPUT_OUTPUT_ATTRIBUTES } from './agent-prism-types';

const getAttributeValue = (
  span: OpenTelemetrySpan,
  key: string,
): string | number | boolean | undefined => {
  const attr = span.attributes.find((a: TraceSpanAttribute) => a.key === key);
  if (!attr) {
    return undefined;
  }
  const { value } = attr;
  if (value.stringValue !== undefined) {
    return value.stringValue;
  }
  if (value.intValue !== undefined) {
    return parseFloat(value.intValue);
  }
  if (value.boolValue !== undefined) {
    return value.boolValue;
  }
  return undefined;
};

const nanoToDate = (nanoString: string): Date => {
  const nanoseconds = BigInt(nanoString);
  const milliseconds = Number(nanoseconds / BigInt(1_000_000));
  return new Date(milliseconds);
};

const getSpanDuration = (span: OpenTelemetrySpan): number => {
  const startNano = BigInt(span.startTimeUnixNano);
  const endNano = BigInt(span.endTimeUnixNano);
  return Number((endNano - startNano) / BigInt(1_000_000));
};

const getSpanInputOutput = (
  span: OpenTelemetrySpan,
): { input?: string; output?: string } => {
  const input = getAttributeValue(span, INPUT_OUTPUT_ATTRIBUTES.INPUT_VALUE);
  const output = getAttributeValue(span, INPUT_OUTPUT_ATTRIBUTES.OUTPUT_VALUE);
  return {
    input: typeof input === 'string' ? input : undefined,
    output: typeof output === 'string' ? output : undefined,
  };
};

const convertRawSpanToTraceSpan = (span: OpenTelemetrySpan): TraceSpan => ({
  id: span.spanId,
  title: span.name,
  status: span.status.code,
  attributes: span.attributes,
  duration: getSpanDuration(span),
  raw: JSON.stringify(span, null, 2),
  startTime: nanoToDate(span.startTimeUnixNano),
  endTime: nanoToDate(span.endTimeUnixNano),
  children: [],
  ...getSpanInputOutput(span),
});

export const convertSpansToTree = (spans: OpenTelemetrySpan[]): TraceSpan[] => {
  const spanMap = new Map<string, TraceSpan>();
  const rootSpans: TraceSpan[] = [];

  spans.forEach((span) => {
    const converted = convertRawSpanToTraceSpan(span);
    spanMap.set(converted.id, converted);
  });

  spans.forEach((span) => {
    const converted = spanMap.get(span.spanId)!;
    const parentSpanId = span.parentSpanId;
    if (parentSpanId) {
      const parent = spanMap.get(parentSpanId);
      if (parent) {
        if (!parent.children) {
          parent.children = [];
        }
        parent.children.push(converted);
      } else {
        rootSpans.push(converted);
      }
    } else {
      rootSpans.push(converted);
    }
  });

  return rootSpans;
};
