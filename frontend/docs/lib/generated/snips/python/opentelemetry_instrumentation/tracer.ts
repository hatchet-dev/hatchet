import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import os\nfrom typing import cast\n\nfrom opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter\nfrom opentelemetry.sdk.resources import SERVICE_NAME, Resource\nfrom opentelemetry.sdk.trace import TracerProvider\nfrom opentelemetry.sdk.trace.export import BatchSpanProcessor\nfrom opentelemetry.trace import NoOpTracerProvider\n\ntrace_provider: TracerProvider | NoOpTracerProvider\n\nif os.getenv(\'CI\', \'false\') == \'true\':\n    trace_provider = NoOpTracerProvider()\nelse:\n    resource = Resource(\n        attributes={\n            SERVICE_NAME: os.getenv(\'HATCHET_CLIENT_OTEL_SERVICE_NAME\', \'test-service\')\n        }\n    )\n\n    headers = dict(\n        [\n            cast(\n                tuple[str, str],\n                tuple(\n                    os.getenv(\n                        \'HATCHET_CLIENT_OTEL_EXPORTER_OTLP_HEADERS\', \'foo=bar\'\n                    ).split(\'=\')\n                ),\n            )\n        ]\n    )\n\n    processor = BatchSpanProcessor(\n        OTLPSpanExporter(\n            endpoint=os.getenv(\n                \'HATCHET_CLIENT_OTEL_EXPORTER_OTLP_ENDPOINT\', \'http://localhost:4317\'\n            ),\n            headers=headers,\n        ),\n    )\n\n    trace_provider = TracerProvider(resource=resource)\n\n    trace_provider.add_span_processor(processor)\n',
  'source': 'out/python/opentelemetry_instrumentation/tracer.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
