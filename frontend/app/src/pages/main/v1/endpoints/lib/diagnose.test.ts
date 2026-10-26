import { diagnoseEndpoint, explainFailure, statusCodeOf } from './diagnose';
import {
  V1ServerlessEndpoint,
  V1ServerlessEndpointCheckStage,
  V1ServerlessEndpointEvent,
  V1ServerlessEndpointEventKind,
  V1ServerlessEndpointKind,
} from '@/lib/api/generated/data-contracts';
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

const NOW = Date.parse('2026-10-22T12:00:00Z');

function endpoint(
  overrides: Omit<Partial<V1ServerlessEndpoint>, 'status'> & {
    status?: Partial<V1ServerlessEndpoint['status']>;
  } = {},
): V1ServerlessEndpoint {
  const { status, ...rest } = overrides;

  return {
    metadata: {
      id: '00000000-0000-0000-0000-000000000001',
      createdAt: '2026-10-22T11:00:00Z',
      updatedAt: '2026-10-22T11:00:00Z',
    },
    tenantId: '00000000-0000-0000-0000-000000000002',
    name: 'ep',
    kind: V1ServerlessEndpointKind.VERCEL,
    healthcheckUrl: 'https://app.vercel.app/api/hatchet/healthcheck',
    triggerUrl: 'https://app.vercel.app/api/hatchet/trigger',
    requestTimeoutSeconds: 300,
    pollIntervalSeconds: 30,
    inlineWaitBudgetMs: 5000,
    labels: {},
    enabled: true,
    ...rest,
    status: {
      registeredActions: ['echo:echo'],
      healthy: true,
      ...status,
    },
  };
}

function event(
  kind: V1ServerlessEndpointEventKind,
  createdAt: string,
  message = '',
  details: V1ServerlessEndpointEvent['details'] = {},
): V1ServerlessEndpointEvent {
  return { id: 1, kind, createdAt, message, details };
}

describe('statusCodeOf', () => {
  it('reads the status from operator error text', () => {
    assert.equal(statusCodeOf('healthcheck returned status 401'), 401);
    assert.equal(
      statusCodeOf('endpoint returned status 503 Service Unavailable'),
      503,
    );
    assert.equal(statusCodeOf('request timed out after 1m0s'), undefined);
  });
});

describe('explainFailure', () => {
  it('blames the secret and Vercel protection on a 401', () => {
    const e = explainFailure(V1ServerlessEndpointKind.VERCEL, {
      deployment: 'oss',
      request: 'healthcheck',
      error: 'healthcheck returned status 401',
    });
    assert.equal(e.title.key, 'failure.auth.title');
    assert.ok(e.steps.some((s) => s.key === 'failure.auth.step.VERCEL'));
  });

  it('mentions Cloudflare Access on a 403 for workers', () => {
    const e = explainFailure(V1ServerlessEndpointKind.CLOUDFLARE_WORKERS, {
      deployment: 'oss',
      request: 'healthcheck',
      statusCode: 403,
    });
    assert.ok(
      e.steps.some((s) => s.key === 'failure.auth.step.CLOUDFLARE_WORKERS'),
    );
  });

  it('names the expected path on a 404', () => {
    const e = explainFailure(V1ServerlessEndpointKind.CLOUDFLARE_WORKERS, {
      deployment: 'oss',
      request: 'healthcheck',
      error: 'healthcheck returned status 404',
    });
    assert.equal(e.steps[0].key, 'failure.notFound.step.healthcheckPath');
    assert.equal(e.steps[0].params?.path, '/hatchet/healthcheck');
  });

  it('points at platform logs on a 5xx', () => {
    const e = explainFailure(V1ServerlessEndpointKind.VERCEL, {
      deployment: 'oss',
      request: 'delivery',
      error: 'endpoint returned status 500 Internal Server Error',
    });
    assert.equal(e.title.key, 'failure.serverError.title');
    assert.equal(e.steps[0].key, 'failure.step.logsCommand');
    assert.equal(e.steps[0].params?.command, 'vercel logs');
  });

  it('recognizes safeclient transport errors', () => {
    const kind = V1ServerlessEndpointKind.GENERIC_HTTP;
    const title = (error: string) =>
      explainFailure(kind, {
        deployment: 'oss',
        request: 'healthcheck',
        error,
      }).title.key;

    assert.equal(
      title(
        'healthcheck request failed: safeclient: could not resolve endpoint host x.example',
      ),
      'failure.dns.title',
    );
    assert.equal(
      title(
        'healthcheck request failed: safeclient: TLS handshake with endpoint host x failed: certificate has expired',
      ),
      'failure.tls.title',
    );
    assert.equal(
      title(
        'healthcheck request failed: safeclient: destination blocked by SSRF policy: endpoint host x resolves to an address the policy does not allow',
      ),
      'failure.blocked.title',
    );
    assert.equal(
      title('healthcheck request failed: context deadline exceeded'),
      'failure.timeout.title',
    );
  });

  it('explains a body that is not a healthcheck', () => {
    const e = explainFailure(V1ServerlessEndpointKind.VERCEL, {
      deployment: 'oss',
      request: 'healthcheck',
      stage: V1ServerlessEndpointCheckStage.PARSE,
      error: 'could not parse healthcheck response: unexpected token',
    });
    assert.equal(e.title.key, 'failure.notHealthcheck.title');
  });

  it('explains a refused catalog', () => {
    const e = explainFailure(V1ServerlessEndpointKind.GENERIC_HTTP, {
      deployment: 'oss',
      request: 'healthcheck',
      error: 'healthcheck advertises an invalid action: bad id',
    });
    assert.equal(e.title.key, 'failure.catalog.title');
  });
});

describe('diagnoseEndpoint', () => {
  it('reports a disabled endpoint first', () => {
    const d = diagnoseEndpoint({
      deployment: 'oss',
      endpoint: endpoint({ enabled: false, status: { healthy: false } }),
      now: NOW,
    });
    assert.equal(d.severity, 'info');
  });

  it('is pending before the first healthcheck, then warns when it takes too long', () => {
    const fresh = endpoint({
      metadata: {
        id: 'x',
        createdAt: '2026-10-22T11:59:50Z',
        updatedAt: '2026-10-22T11:59:50Z',
      },
      status: { healthy: undefined },
    });
    assert.equal(
      diagnoseEndpoint({
        deployment: 'oss',
        endpoint: fresh,
        now: NOW,
      }).severity,
      'pending',
    );

    const stale = endpoint({ status: { healthy: undefined } });
    const oss = diagnoseEndpoint({
      deployment: 'oss',
      endpoint: stale,
      now: NOW,
    });
    assert.equal(oss.severity, 'warning');
    assert.equal(oss.title.key, 'diagnosis.notPolled.title');
    assert.equal(oss.steps[0].key, 'diagnosis.notPolled.steps.oss');

    const cloud = diagnoseEndpoint({
      deployment: 'cloud',
      endpoint: stale,
      now: NOW,
    });
    assert.equal(cloud.steps[0].key, 'diagnosis.notPolled.steps.cloud');
  });

  it('explains early failures before the endpoint turns unhealthy', () => {
    const d = diagnoseEndpoint({
      deployment: 'oss',
      endpoint: endpoint({ status: { healthy: undefined } }),
      events: [
        event(
          V1ServerlessEndpointEventKind.HEALTHCHECK_FAILED,
          '2026-10-22T11:59:00Z',
          'healthcheck returned status 401',
          { statusCode: 401 },
        ),
      ],
      now: NOW,
    });
    assert.equal(d.severity, 'error');
    assert.equal(d.title.key, 'failure.auth.title');
  });

  it('explains an unhealthy endpoint from its status error', () => {
    const d = diagnoseEndpoint({
      deployment: 'oss',
      endpoint: endpoint({
        status: { healthy: false, error: 'healthcheck returned status 502' },
      }),
      now: NOW,
    });
    assert.equal(d.title.key, 'failure.serverError.title');
  });

  it('flags recent delivery failures on a healthy endpoint', () => {
    const d = diagnoseEndpoint({
      deployment: 'oss',
      endpoint: endpoint(),
      events: [
        event(
          V1ServerlessEndpointEventKind.DELIVERY_FAILED,
          '2026-10-22T11:58:00Z',
          'request timed out after 5m0s',
        ),
      ],
      now: NOW,
    });
    assert.equal(d.severity, 'warning');
    assert.equal(d.title.key, 'diagnosis.deliveriesFailing.title');
  });

  it('ignores old delivery failures', () => {
    const d = diagnoseEndpoint({
      deployment: 'oss',
      endpoint: endpoint(),
      events: [
        event(
          V1ServerlessEndpointEventKind.DELIVERY_FAILED,
          '2026-10-22T10:00:00Z',
          'endpoint returned status 500',
        ),
      ],
      now: NOW,
    });
    assert.equal(d.severity, 'healthy');
  });

  it('warns about failing healthchecks that have not flipped the status yet', () => {
    const d = diagnoseEndpoint({
      deployment: 'oss',
      endpoint: endpoint(),
      events: [
        event(V1ServerlessEndpointEventKind.HEALTHY, '2026-10-22T11:00:00Z'),
        event(
          V1ServerlessEndpointEventKind.HEALTHCHECK_FAILED,
          '2026-10-22T11:59:00Z',
          'healthcheck returned status 503',
        ),
      ],
      now: NOW,
    });
    assert.equal(d.severity, 'warning');
    assert.equal(d.title.key, 'diagnosis.healthchecksFailing.title');
  });

  it('stops warning once a short failure streak must have ended', () => {
    // 30s polls: a streak still going on without flipping the status is under two minutes old
    const d = diagnoseEndpoint({
      deployment: 'oss',
      endpoint: endpoint(),
      events: [
        event(V1ServerlessEndpointEventKind.HEALTHY, '2026-10-22T11:00:00Z'),
        event(
          V1ServerlessEndpointEventKind.HEALTHCHECK_FAILED,
          '2026-10-22T11:30:00Z',
          'healthcheck returned status 503',
        ),
      ],
      now: NOW,
    });
    assert.equal(d.severity, 'healthy');
  });

  it('takes the status flipping back to healthy as a recovery', () => {
    const d = diagnoseEndpoint({
      deployment: 'oss',
      endpoint: endpoint({
        pollIntervalSeconds: 3600,
        status: { healthy: true, changedAt: '2026-10-22T11:59:30Z' },
      }),
      events: [
        event(
          V1ServerlessEndpointEventKind.HEALTHCHECK_FAILED,
          '2026-10-22T11:59:00Z',
          'healthcheck returned status 503',
        ),
      ],
      now: NOW,
    });
    assert.equal(d.severity, 'healthy');
  });

  it('only uses event details that belong to the status error', () => {
    const parseFailure = event(
      V1ServerlessEndpointEventKind.HEALTHCHECK_FAILED,
      '2026-10-22T11:59:00Z',
      'bad body',
      { stage: V1ServerlessEndpointCheckStage.PARSE },
    );
    const unhealthy = (error: string) =>
      diagnoseEndpoint({
        deployment: 'oss',
        endpoint: endpoint({ status: { healthy: false, error } }),
        events: [parseFailure],
        now: NOW,
      }).title.key;

    assert.equal(unhealthy('bad body'), 'failure.notHealthcheck.title');
    // a later failure changed the status error; the old event's stage does not apply
    assert.equal(unhealthy('something else'), 'failure.unknown.title');
    assert.equal(
      unhealthy('healthcheck returned status 503'),
      'failure.serverError.title',
    );
  });

  it('warns about a healthy endpoint that serves nothing', () => {
    const d = diagnoseEndpoint({
      deployment: 'oss',
      endpoint: endpoint({ status: { registeredActions: [] } }),
      now: NOW,
    });
    assert.equal(d.title.key, 'diagnosis.noActions.title');
  });
});

describe('cloud and self-hosted advice', () => {
  it('only gives self-hosted steps for a missing operator token', () => {
    const failure = {
      request: 'healthcheck' as const,
      error: 'no token for tenant 707d0855',
    };
    const cloud = explainFailure(V1ServerlessEndpointKind.GENERIC_HTTP, {
      ...failure,
      deployment: 'cloud',
    });
    const oss = explainFailure(V1ServerlessEndpointKind.GENERIC_HTTP, {
      ...failure,
      deployment: 'oss',
    });

    assert.equal(cloud.detail.key, 'failure.operatorToken.detail.cloud');
    assert.deepEqual(cloud.steps, []);
    assert.equal(oss.detail.key, 'failure.operatorToken.detail.oss');
    assert.equal(oss.steps[0].key, 'failure.operatorToken.steps.oss');
  });

  it('suggests insecure destinations for local development only on self-hosted', () => {
    const error =
      'healthcheck request failed: safeclient: destination blocked by SSRF policy';
    const steps = (deployment: 'cloud' | 'oss') =>
      explainFailure(V1ServerlessEndpointKind.GENERIC_HTTP, {
        request: 'healthcheck',
        error,
        deployment,
      }).steps.map((s) => s.key);

    assert.ok(!steps('cloud').includes('failure.blocked.step.oss'));
    assert.ok(steps('oss').includes('failure.blocked.step.oss'));
  });
});
