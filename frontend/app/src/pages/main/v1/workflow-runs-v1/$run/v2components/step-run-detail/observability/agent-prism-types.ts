// Types

export type TraceRecord = {
  id: string;
  name: string;
  spansCount: number;
  durationMs: number;
  agentDescription: string;
  totalCost?: number;
  totalTokens?: number;
  startTime?: number;
};

export type TraceSpanStatus = 'success' | 'error' | 'pending' | 'warning';

export type InputOutputData = {
  input?: string;
  output?: string;
};

export type TraceSpan<TMetadata = Record<string, unknown>> = InputOutputData & {
  id: string;
  title: string;
  startTime: Date;
  endTime: Date;
  duration: number;
  type: TraceSpanCategory;
  raw: string;
  attributes?: TraceSpanAttribute[];
  children?: TraceSpan<TMetadata>[];
  status: TraceSpanStatus;
  cost?: number;
  tokensCount?: number;
  metadata?: TMetadata;
};

export type TraceSpanCategory =
  | 'llm_call'
  | 'tool_execution'
  | 'agent_invocation'
  | 'chain_operation'
  | 'retrieval'
  | 'embedding'
  | 'create_agent'
  | 'span'
  | 'event'
  | 'guardrail'
  | 'unknown';

export type TraceSpanAttribute = {
  key: string;
  value: TraceSpanAttributeValue;
};

export type TraceSpanAttributeValue = {
  stringValue?: string;
  intValue?: string;
  boolValue?: boolean;
};

// OpenTelemetry types

export type OpenTelemetryDocument = {
  resourceSpans: OpenTelemetryResourceSpan[];
};

export type OpenTelemetryResourceSpan = {
  resource: OpenTelemetryResource;
  scopeSpans: OpenTelemetryScopeSpan[];
  schemaUrl?: string;
};

export type OpenTelemetryResource = {
  attributes: TraceSpanAttribute[];
};

export type OpenTelemetryScopeSpan = {
  scope: OpenTelemetryScope;
  spans: OpenTelemetrySpan[];
  schemaUrl?: string;
};

export type OpenTelemetryScope = {
  name: string;
  version?: string;
};

export type OpenTelemetrySpan = {
  traceId: string;
  spanId: string;
  parentSpanId?: string;
  name: string;
  kind: OpenTelemetrySpanKind;
  startTimeUnixNano: string;
  endTimeUnixNano: string;
  attributes: TraceSpanAttribute[];
  status: OpenTelemetryStatus;
  flags: number;
  events?: OpenTelemetryEvent[];
  traceState?: string;
  droppedAttributesCount?: number;
  droppedEventsCount?: number;
  droppedLinksCount?: number;
  links?: OpenTelemetryLink[];
};

export type OpenTelemetryEvent = {
  timeUnixNano: string;
  name: string;
  attributes?: TraceSpanAttribute[];
  droppedAttributesCount?: number;
};

export type OpenTelemetryLink = {
  traceId: string;
  spanId: string;
  traceState?: string;
  attributes?: TraceSpanAttribute[];
  droppedAttributesCount?: number;
};

export type OpenTelemetryStatus = {
  code?: OpenTelemetryStatusCode;
  message?: string;
};

export type OpenTelemetrySpanKind =
  | 'SPAN_KIND_INTERNAL'
  | 'SPAN_KIND_SERVER'
  | 'SPAN_KIND_CLIENT'
  | 'SPAN_KIND_PRODUCER'
  | 'SPAN_KIND_CONSUMER';

export type OpenTelemetryStatusCode =
  | 'STATUS_CODE_OK'
  | 'STATUS_CODE_ERROR'
  | 'STATUS_CODE_UNSET';

export type OpenTelemetryStandard =
  | 'opentelemetry_genai'
  | 'openinference'
  | 'standard';

// Constants

export const OPENTELEMETRY_GENAI_ATTRIBUTES = {
  OPERATION_NAME: 'gen_ai.operation.name',
  SYSTEM: 'gen_ai.system',
  MODEL: 'gen_ai.request.model',
  AGENT_NAME: 'gen_ai.agent.name',
  TOOL_NAME: 'gen_ai.tool.name',
  USAGE_INPUT_TOKENS: 'gen_ai.usage.input_tokens',
  USAGE_OUTPUT_TOKENS: 'gen_ai.usage.output_tokens',
  USAGE_TOTAL_TOKENS: 'gen_ai.usage.total_tokens',
  USAGE_COST: 'gen_ai.usage.cost',
  USAGE_INPUT_COST: 'gen_ai.usage.input_cost',
  USAGE_OUTPUT_COST: 'gen_ai.usage.output_cost',
  REQUEST_TEMPERATURE: 'gen_ai.request.temperature',
  REQUEST_PROMPT: 'gen_ai.request.prompt',
  RESPONSE_TEXT: 'gen_ai.response.text',
} as const;

export const OPENINFERENCE_ATTRIBUTES = {
  SPAN_KIND: 'openinference.span.kind',
  LLM_MODEL: 'llm.model_name',
  INPUT_MESSAGES: 'llm.input_messages',
  RETRIEVAL_DOCUMENTS: 'retrieval.documents',
  EMBEDDING_MODEL: 'embedding.model_name',
} as const;

export const STANDARD_OPENTELEMETRY_ATTRIBUTES = {
  HTTP_METHOD: 'http.method',
  HTTP_URL: 'http.url',
  DB_SYSTEM: 'db.system',
  DB_OPERATION: 'db.operation.name',
  DB_COLLECTION: 'db.collection.name',
  DB_QUERY_TEXT: 'db.query.text',
  FUNCTION_NAME: 'function.name',
} as const;

export const OPENTELEMETRY_GENAI_MAPPINGS: Record<string, TraceSpanCategory> = {
  chat: 'llm_call',
  generate_content: 'llm_call',
  text_completion: 'llm_call',
  execute_tool: 'tool_execution',
  invoke_agent: 'agent_invocation',
  create_agent: 'create_agent',
  embeddings: 'embedding',
};

export const OPENINFERENCE_MAPPINGS: Record<string, TraceSpanCategory> = {
  LLM: 'llm_call',
  TOOL: 'tool_execution',
  CHAIN: 'chain_operation',
  AGENT: 'agent_invocation',
  RETRIEVER: 'retrieval',
  EMBEDDING: 'embedding',
};

export const STANDARD_OPENTELEMETRY_PATTERNS = {
  HTTP_KEYWORDS: [] as string[],
  DATABASE_KEYWORDS: [] as string[],
  FUNCTION_KEYWORDS: ['tool', 'function'],
  LLM_KEYWORDS: ['openai', 'anthropic', 'gpt', 'claude'],
  CHAIN_KEYWORDS: ['chain', 'workflow', 'langchain'],
  AGENT_KEYWORDS: ['agent'],
  RETRIEVAL_KEYWORDS: ['pinecone', 'chroma', 'retrieval', 'vector', 'search'],
} as const;

export const INPUT_OUTPUT_ATTRIBUTES = {
  INPUT_VALUE: 'input.value',
  OUTPUT_VALUE: 'output.value',
} as const;
