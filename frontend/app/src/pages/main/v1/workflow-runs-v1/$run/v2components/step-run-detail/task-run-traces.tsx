import { cloudApi } from '@/lib/api/api';
import { OtelSpan } from '@/lib/api/generated/cloud/data-contracts';
import { Loading } from '@/components/v1/ui/loading';
import { useQuery } from '@tanstack/react-query';

export function TaskRunTraces({
  workflowRunId,
}: {
  workflowRunId: string;
}) {
  const tracesQuery = useQuery({
    queryKey: ['cloud:otel-traces', workflowRunId],
    queryFn: async () => {
      const res = await cloudApi.otelTracesList(workflowRunId);
      return res.data;
    },
  });

  if (tracesQuery.isLoading) {
    return <Loading />;
  }

  if (tracesQuery.isError) {
    return (
      <div className="py-4 text-sm text-muted-foreground">
        Failed to load traces.
      </div>
    );
  }

  const spans = tracesQuery.data?.rows;

  if (!spans || spans.length === 0) {
    return (
      <div className="py-4 text-sm text-muted-foreground">
        No traces found for this workflow run.
      </div>
    );
  }

  return (
    <div className="my-4 flex flex-col gap-2">
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="border-b text-muted-foreground">
            <th className="pb-2 pr-4 font-medium">Span Name</th>
            <th className="pb-2 pr-4 font-medium">Kind</th>
            <th className="pb-2 pr-4 font-medium">Service</th>
            <th className="pb-2 pr-4 font-medium">Status</th>
            <th className="pb-2 pr-4 font-medium">Duration</th>
            <th className="pb-2 font-medium">Time</th>
          </tr>
        </thead>
        <tbody>
          {spans.map((span) => (
            <SpanRow key={span.span_id} span={span} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function formatNanoDuration(nanos: number): string {
  if (nanos < 1_000) {
    return `${nanos}ns`;
  }
  if (nanos < 1_000_000) {
    return `${(nanos / 1_000).toFixed(1)}µs`;
  }
  if (nanos < 1_000_000_000) {
    return `${(nanos / 1_000_000).toFixed(1)}ms`;
  }
  return `${(nanos / 1_000_000_000).toFixed(2)}s`;
}

function SpanRow({ span }: { span: OtelSpan }) {
  return (
    <tr className="border-b last:border-0">
      <td className="py-2 pr-4 font-mono">{span.span_name}</td>
      <td className="py-2 pr-4">{span.span_kind}</td>
      <td className="py-2 pr-4">{span.service_name}</td>
      <td className="py-2 pr-4">
        <span
          className={
            span.status_code === 'ERROR'
              ? 'text-red-500'
              : span.status_code === 'OK'
                ? 'text-green-500'
                : 'text-muted-foreground'
          }
        >
          {span.status_code}
          {span.status_message ? `: ${span.status_message}` : ''}
        </span>
      </td>
      <td className="py-2 pr-4 font-mono">
        {formatNanoDuration(span.duration)}
      </td>
      <td className="py-2">
        {new Date(span.created_at).toLocaleString()}
      </td>
    </tr>
  );
}
