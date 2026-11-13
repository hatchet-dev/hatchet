import {
  FeatureFlagEvaluationResult,
  Tenant,
  TenantMember,
  TenantMemberRole,
  TenantResourcePolicy,
  TenantVersion,
  User,
  V1TaskPointMetrics,
  V1TaskRunMetrics,
  V1TaskStatus,
  V1TaskSummary,
  V1TaskSummaryList,
  V1WorkflowType,
  Workflow,
  WorkflowList,
} from '@/lib/api';
import { delay, http, HttpResponse } from 'msw';

export const STORY_TENANT_ID = '00000000-0000-4000-8000-000000000001';

const meta = (id: string, createdAt: string) => ({
  id,
  createdAt,
  updatedAt: createdAt,
});

const epoch = '2026-01-01T00:00:00Z';

const user: User = {
  metadata: meta('00000000-0000-4000-8000-0000000000a1', epoch),
  name: 'Storybook',
  email: 'storybook@example.com',
  emailVerified: true,
};

const tenant: Tenant = {
  metadata: meta(STORY_TENANT_ID, epoch),
  name: 'storybook',
  slug: 'storybook',
  version: TenantVersion.V1,
  dataRetentionPeriod: '720h',
};

const membership: TenantMember = {
  metadata: meta('00000000-0000-4000-8000-0000000000b1', epoch),
  user: { email: user.email, name: user.name },
  role: TenantMemberRole.OWNER,
  tenant,
};

const WORKFLOW_NAMES = [
  'process-order',
  'send-invoice',
  'sync-inventory',
  'nightly-report',
  'resize-image',
];

const workflows: Workflow[] = WORKFLOW_NAMES.map((name, i) => ({
  metadata: meta(`00000000-0000-4000-8000-0000000001${i}0`, epoch),
  name,
  tenantId: STORY_TENANT_ID,
}));

// Weighted so the table and counts look like a real tenant: mostly
// successes, a few failures, a handful in flight.
const STATUS_CYCLE: V1TaskStatus[] = [
  V1TaskStatus.COMPLETED,
  V1TaskStatus.COMPLETED,
  V1TaskStatus.COMPLETED,
  V1TaskStatus.FAILED,
  V1TaskStatus.COMPLETED,
  V1TaskStatus.RUNNING,
  V1TaskStatus.COMPLETED,
  V1TaskStatus.QUEUED,
  V1TaskStatus.COMPLETED,
  V1TaskStatus.CANCELLED,
];

const MINUTE = 60 * 1000;

// Dates are relative to load time so runs always fall inside the default
// 1-day window and the relative times stay readable.
function buildRuns(count: number): V1TaskSummary[] {
  const now = Date.now();

  return Array.from({ length: count }, (_, i) => {
    const status = STATUS_CYCLE[i % STATUS_CYCLE.length];
    const workflow = workflows[i % workflows.length];
    const created = now - (i * 17 + 2) * MINUTE;
    const started =
      status === V1TaskStatus.QUEUED ? undefined : created + 4_000 + i * 150;
    const finished =
      status === V1TaskStatus.QUEUED || status === V1TaskStatus.RUNNING
        ? undefined
        : (started ?? created) + 1_500 + (i % 7) * 2_300;
    const id = `00000000-0000-4000-8000-${String(i + 1).padStart(12, '0')}`;
    const iso = (t: number) => new Date(t).toISOString();

    return {
      metadata: meta(id, iso(created)),
      createdAt: iso(created),
      displayName: `${workflow.name}-${1794320000 + i * 37}`,
      input: {},
      output: {},
      numSpawnedChildren: 0,
      status,
      // Every other running run is evicted, so both sub-filters have rows.
      isEvicted:
        status === V1TaskStatus.RUNNING && Math.floor(i / 10) % 2 === 1,
      startedAt: started ? iso(started) : undefined,
      finishedAt: finished ? iso(finished) : undefined,
      duration: started && finished ? finished - started : undefined,
      errorMessage:
        status === V1TaskStatus.FAILED ? 'upstream returned 503' : undefined,
      additionalMetadata: {
        env: i % 3 === 0 ? 'staging' : 'prod',
        customer_id: `cus_${8800 + (i % 9)}`,
      },
      taskExternalId: id,
      taskId: i + 1,
      taskInsertedAt: iso(created),
      tenantId: STORY_TENANT_ID,
      type: V1WorkflowType.TASK,
      workflowId: workflow.metadata.id,
      workflowName: workflow.name,
      workflowRunExternalId: id,
    };
  });
}

function buildPointMetrics(runs: V1TaskSummary[]): V1TaskPointMetrics {
  const bucket = 30 * MINUTE;
  const now = Date.now();
  const start = now - 24 * 60 * MINUTE;
  const results = Array.from({ length: 48 }, (_, i) => ({
    time: new Date(start + i * bucket).toISOString(),
    SUCCEEDED: 0,
    FAILED: 0,
  }));

  for (const run of runs) {
    const idx = Math.floor((Date.parse(run.createdAt) - start) / bucket);
    if (idx < 0 || idx >= results.length) {
      continue;
    }
    if (run.status === V1TaskStatus.COMPLETED) {
      results[idx].SUCCEEDED += 1;
    } else if (run.status === V1TaskStatus.FAILED) {
      results[idx].FAILED += 1;
    }
  }

  return { results };
}

function statusMetrics(runs: V1TaskSummary[]): V1TaskRunMetrics {
  return Object.values(V1TaskStatus).map((status) => {
    const matching = runs.filter((r) => r.status === status);
    return {
      status,
      count: matching.length,
      runningDetailCount:
        status === V1TaskStatus.RUNNING
          ? {
              onWorker: matching.filter((r) => !r.isEvicted).length,
              evicted: matching.filter((r) => r.isEvicted).length,
            }
          : undefined,
    };
  });
}

// Applies the filters the sidebar sends most often, so clicking around in
// the story changes the table. Other params are accepted and ignored.
function filterRuns(runs: V1TaskSummary[], url: URL) {
  const statuses = url.searchParams.getAll('statuses');
  const workflowIds = url.searchParams.getAll('workflow_ids');

  return runs.filter(
    (r) =>
      (statuses.length === 0 || statuses.includes(r.status)) &&
      (workflowIds.length === 0 || workflowIds.includes(r.workflowId)),
  );
}

const base = '*/api/v1';
const tenantPath = `${base}/stable/tenants/:tenant`;

function sessionHandlers() {
  return [
    // A 404 is how the app detects a self-hosted install, which keeps
    // requests on the same origin instead of exchanging tokens for a shard.
    http.get(`${base}/control-plane/metadata`, () =>
      HttpResponse.json({}, { status: 404 }),
    ),
    http.get(`${base}/users/current`, () => HttpResponse.json(user)),
    http.get(`${base}/users/memberships`, () =>
      HttpResponse.json({ rows: [membership] }),
    ),
    // The stories show the table as it looks with the filter sidebar flag on.
    http.get(`${base}/tenants/:tenant/feature-flags`, () =>
      HttpResponse.json({
        isEnabled: true,
      } satisfies FeatureFlagEvaluationResult),
    ),
    http.get(`${base}/tenants/:tenant/resource-policy`, () =>
      HttpResponse.json({ limits: [] } satisfies TenantResourcePolicy),
    ),
    http.get(`${base}/tenants/:tenant/workflows`, () =>
      HttpResponse.json({ rows: workflows } satisfies WorkflowList),
    ),
  ];
}

export function runsHandlers(runCount: number) {
  const runs = buildRuns(runCount);

  return [
    ...sessionHandlers(),
    http.get(`${tenantPath}/workflow-runs`, ({ request }) => {
      const url = new URL(request.url);
      const filtered = filterRuns(runs, url);
      const offset = Number(url.searchParams.get('offset') ?? 0);
      const limit = Number(url.searchParams.get('limit') ?? 50);

      return HttpResponse.json({
        rows: filtered.slice(offset, offset + limit),
        pagination: {
          current_page: Math.floor(offset / limit) + 1,
          num_pages: Math.max(1, Math.ceil(filtered.length / limit)),
        },
      } satisfies V1TaskSummaryList);
    }),
    http.get(`${tenantPath}/workflow-runs/count`, ({ request }) =>
      HttpResponse.json({
        count: filterRuns(runs, new URL(request.url)).length,
      }),
    ),
    http.get(`${tenantPath}/task-metrics`, () =>
      HttpResponse.json(statusMetrics(runs)),
    ),
    http.get(`${tenantPath}/task-point-metrics`, () =>
      HttpResponse.json(buildPointMetrics(runs)),
    ),
  ];
}

export function loadingHandlers() {
  return [
    ...sessionHandlers(),
    http.get(`${tenantPath}/*`, async () => {
      await delay('infinite');
      return HttpResponse.json({});
    }),
  ];
}
