import type {
  OpenTelemetrySpan,
  TraceSpan,
  TraceSpanCategory,
  TraceSpanStatus,
  TraceSpanAttribute,
} from './agent-prism-types';
import {
  INPUT_OUTPUT_ATTRIBUTES,
  OPENTELEMETRY_GENAI_ATTRIBUTES,
  OPENTELEMETRY_GENAI_MAPPINGS,
  OPENINFERENCE_ATTRIBUTES,
  OPENINFERENCE_MAPPINGS,
  STANDARD_OPENTELEMETRY_ATTRIBUTES,
  STANDARD_OPENTELEMETRY_PATTERNS,
} from './agent-prism-types';

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

const getSpanStatus = (span: OpenTelemetrySpan): TraceSpanStatus => {
  switch (span.status.code) {
    case 'STATUS_CODE_OK':
      return 'success';
    case 'STATUS_CODE_ERROR':
      return 'error';
    default:
      return 'warning';
  }
};

const getSpanTitle = (span: OpenTelemetrySpan): string => {
  const { name } = span;

  const model = getAttributeValue(span, OPENTELEMETRY_GENAI_ATTRIBUTES.MODEL);
  if (model) {
    return `${model} - ${name}`;
  }

  const collection = getAttributeValue(
    span,
    STANDARD_OPENTELEMETRY_ATTRIBUTES.DB_COLLECTION,
  );
  const operation = getAttributeValue(
    span,
    STANDARD_OPENTELEMETRY_ATTRIBUTES.DB_OPERATION,
  );
  if (collection && operation) {
    return `${collection} - ${operation}`;
  }

  const method = getAttributeValue(
    span,
    STANDARD_OPENTELEMETRY_ATTRIBUTES.HTTP_METHOD,
  );
  const url = getAttributeValue(
    span,
    STANDARD_OPENTELEMETRY_ATTRIBUTES.HTTP_URL,
  );
  if (method && url) {
    return `${method} ${url}`;
  }

  return name;
};

const getSpanCategory = (span: OpenTelemetrySpan): TraceSpanCategory => {
  const hasGenAI =
    getAttributeValue(span, OPENTELEMETRY_GENAI_ATTRIBUTES.OPERATION_NAME) ||
    getAttributeValue(span, OPENTELEMETRY_GENAI_ATTRIBUTES.SYSTEM);
  if (hasGenAI) {
    const opName = getAttributeValue(
      span,
      OPENTELEMETRY_GENAI_ATTRIBUTES.OPERATION_NAME,
    );
    if (typeof opName === 'string') {
      const category =
        OPENTELEMETRY_GENAI_MAPPINGS[
          opName as keyof typeof OPENTELEMETRY_GENAI_MAPPINGS
        ];
      if (category) {
        return category;
      }
    }
    return categorizeStandard(span);
  }

  const hasOpenInference =
    getAttributeValue(span, OPENINFERENCE_ATTRIBUTES.SPAN_KIND) ||
    getAttributeValue(span, OPENINFERENCE_ATTRIBUTES.LLM_MODEL);
  if (hasOpenInference) {
    const spanKind = getAttributeValue(
      span,
      OPENINFERENCE_ATTRIBUTES.SPAN_KIND,
    );
    if (typeof spanKind === 'string') {
      const category =
        OPENINFERENCE_MAPPINGS[spanKind as keyof typeof OPENINFERENCE_MAPPINGS];
      if (category) {
        return category;
      }
    }
    return categorizeStandard(span);
  }

  return categorizeStandard(span);
};

const categorizeStandard = (span: OpenTelemetrySpan): TraceSpanCategory => {
  const name = span.name.toLowerCase();

  if (
    STANDARD_OPENTELEMETRY_PATTERNS.LLM_KEYWORDS.some((kw: string) =>
      name.includes(kw),
    )
  ) {
    return 'llm_call';
  }
  if (
    STANDARD_OPENTELEMETRY_PATTERNS.AGENT_KEYWORDS.some((kw: string) =>
      name.includes(kw),
    )
  ) {
    return 'agent_invocation';
  }
  if (
    STANDARD_OPENTELEMETRY_PATTERNS.CHAIN_KEYWORDS.some((kw: string) =>
      name.includes(kw),
    )
  ) {
    return 'chain_operation';
  }
  if (
    STANDARD_OPENTELEMETRY_PATTERNS.RETRIEVAL_KEYWORDS.some((kw: string) =>
      name.includes(kw),
    )
  ) {
    return 'retrieval';
  }
  if (
    STANDARD_OPENTELEMETRY_PATTERNS.FUNCTION_KEYWORDS.some((kw: string) =>
      name.includes(kw),
    ) ||
    getAttributeValue(span, STANDARD_OPENTELEMETRY_ATTRIBUTES.FUNCTION_NAME) !==
      undefined
  ) {
    return 'tool_execution';
  }
  if (
    getAttributeValue(span, STANDARD_OPENTELEMETRY_ATTRIBUTES.HTTP_METHOD) !==
    undefined
  ) {
    return 'tool_execution';
  }
  if (
    getAttributeValue(span, STANDARD_OPENTELEMETRY_ATTRIBUTES.DB_SYSTEM) !==
    undefined
  ) {
    return 'tool_execution';
  }

  return 'unknown';
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

const getSpanTokensCount = (span: OpenTelemetrySpan): number => {
  const totalTokens = getAttributeValue(
    span,
    OPENTELEMETRY_GENAI_ATTRIBUTES.USAGE_TOTAL_TOKENS,
  );
  const inputTokens = getAttributeValue(
    span,
    OPENTELEMETRY_GENAI_ATTRIBUTES.USAGE_INPUT_TOKENS,
  );
  const outputTokens = getAttributeValue(
    span,
    OPENTELEMETRY_GENAI_ATTRIBUTES.USAGE_OUTPUT_TOKENS,
  );
  if (typeof totalTokens === 'number') {
    return totalTokens;
  }
  return (
    (typeof inputTokens === 'number' ? inputTokens : 0) +
    (typeof outputTokens === 'number' ? outputTokens : 0)
  );
};

const getSpanCost = (span: OpenTelemetrySpan): number => {
  const inputCost = getAttributeValue(
    span,
    OPENTELEMETRY_GENAI_ATTRIBUTES.USAGE_INPUT_COST,
  );
  const outputCost = getAttributeValue(
    span,
    OPENTELEMETRY_GENAI_ATTRIBUTES.USAGE_OUTPUT_COST,
  );
  let totalCost = 0;
  if (typeof inputCost === 'number') {
    totalCost += inputCost;
  }
  if (typeof outputCost === 'number') {
    totalCost += outputCost;
  }
  if (totalCost === 0) {
    const fallbackCost = getAttributeValue(span, 'gen_ai.usage.cost');
    if (typeof fallbackCost === 'number') {
      totalCost = fallbackCost;
    }
  }
  return totalCost;
};

const convertRawSpanToTraceSpan = (span: OpenTelemetrySpan): TraceSpan => ({
  id: span.spanId,
  title: getSpanTitle(span),
  type: getSpanCategory(span),
  status: getSpanStatus(span),
  attributes: span.attributes,
  duration: getSpanDuration(span),
  tokensCount: getSpanTokensCount(span),
  raw: JSON.stringify(span, null, 2),
  cost: getSpanCost(span),
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
