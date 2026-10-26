import { CopyKey, CopyParams, CopyRef, ref } from './copy';
import { PLATFORMS } from './platforms';
import {
  V1ServerlessEndpoint,
  V1ServerlessEndpointCheckStage,
  V1ServerlessEndpointEvent,
  V1ServerlessEndpointEventKind,
  V1ServerlessEndpointKind,
} from '@/lib/api/generated/data-contracts';

// Hatchet Cloud, or a self-hosted instance; some advice only applies to one
export type Deployment = 'cloud' | 'oss';

export type DiagnosisSeverity =
  'healthy' | 'info' | 'pending' | 'warning' | 'error';

export type Diagnosis = {
  severity: DiagnosisSeverity;
  title: CopyRef;
  // the failure behind a summarizing title, shown under it
  cause?: CopyRef;
  // sentences, shown as one paragraph; may be empty
  detail: CopyRef[];
  // concrete things to check or do, most likely first; a ref may hold a list
  steps: CopyRef[];
};

export type FailureExplanation = {
  title: CopyRef;
  detail: CopyRef;
  steps: CopyRef[];
};

type Failure = {
  error?: string;
  statusCode?: number;
  stage?: V1ServerlessEndpointCheckStage;
  // what failed: the healthcheck the operator polls, or a task delivery
  request: 'healthcheck' | 'delivery';
  deployment: Deployment;
};

// the failures copy.mdx explains with a title and a detail
type FailureId = {
  [K in CopyKey]: K extends `failure.${infer Id}.title` ? Id : never;
}[CopyKey];

// deliveries that failed within this window count as "currently failing"
const RECENT_DELIVERY_FAILURE_MS = 15 * 60 * 1000;

export function statusCodeOf(error?: string): number | undefined {
  const match = error?.match(/status (\d{3})/);
  return match ? Number(match[1]) : undefined;
}

function includesAny(haystack: string, needles: string[]): boolean {
  return needles.some((n) => haystack.includes(n));
}

/**
 * Explains why one request to an endpoint failed, from its error text and,
 * when known, the HTTP status and the healthcheck stage it reached.
 */
export function explainFailure(
  kind: V1ServerlessEndpointKind,
  failure: Failure,
): FailureExplanation {
  const platform = PLATFORMS[kind];
  const lower = (failure.error ?? '').toLowerCase();
  const status = failure.statusCode ?? statusCodeOf(failure.error);
  const what = ref(
    failure.request === 'healthcheck'
      ? 'request.healthcheck'
      : 'request.delivery',
  );
  const platformLabel = ref(`platform.${kind}.label`);
  const logsStep = platform.logsCommand
    ? ref('failure.step.logsCommand', { command: platform.logsCommand })
    : ref('failure.step.logs');

  const explain = (
    id: FailureId,
    steps: CopyRef[],
    params?: CopyParams,
  ): FailureExplanation => ({
    title: ref(`failure.${id}.title`, params),
    detail: ref(`failure.${id}.detail` as CopyKey, params),
    steps,
  });

  if (lower.includes('no token for tenant')) {
    return {
      title: ref('failure.operatorToken.title'),
      detail: ref(`failure.operatorToken.detail.${failure.deployment}`),
      steps:
        failure.deployment === 'oss'
          ? [ref('failure.operatorToken.steps.oss')]
          : [],
    };
  }

  if (includesAny(lower, ['signing secret', 'decrypt'])) {
    return explain('secretUnusable', [ref('failure.secretUnusable.steps')]);
  }

  if (
    includesAny(lower, [
      'blocked by ssrf policy',
      'scheme not allowed',
      'port not allowed',
    ])
  ) {
    return explain(
      'blocked',
      failure.deployment === 'oss'
        ? [ref('failure.blocked.steps'), ref('failure.blocked.step.oss')]
        : [ref('failure.blocked.steps')],
    );
  }

  if (includesAny(lower, ['could not resolve endpoint host', 'no such host'])) {
    return explain('dns', [ref('failure.dns.steps')], { what });
  }

  if (includesAny(lower, ['tls handshake', 'x509', 'certificate'])) {
    return explain('tls', [ref('failure.tls.steps')], { what });
  }

  if (
    includesAny(lower, [
      'timed out',
      'deadline exceeded',
      'timeout',
      'context canceled',
    ])
  ) {
    return {
      title: ref('failure.timeout.title', { what }),
      detail: ref(`failure.timeout.${failure.request}.detail`),
      steps: [logsStep, ref(`failure.timeout.${failure.request}.step`)],
    };
  }

  if (
    includesAny(lower, [
      'could not connect to endpoint host',
      'failed while reading the response',
      'connection refused',
    ])
  ) {
    return explain('connection', [ref('failure.connection.step'), logsStep], {
      what,
    });
  }

  if (status !== undefined) {
    if (status === 401 || status === 403) {
      return explain(
        'auth',
        [ref('failure.auth.steps'), ref(`failure.auth.step.${kind}`)],
        { what, status },
      );
    }

    if (status === 404 || status === 405) {
      const expectedPath =
        failure.request === 'healthcheck'
          ? platform.healthcheckPath
          : platform.triggerPath;

      return explain(
        'notFound',
        [
          expectedPath
            ? ref(
                failure.request === 'healthcheck'
                  ? 'failure.notFound.step.healthcheckPath'
                  : 'failure.notFound.step.triggerPath',
                { platform: platformLabel, path: expectedPath },
              )
            : ref('failure.notFound.step.path'),
          ref('failure.notFound.step.redeploy'),
        ],
        { what, status },
      );
    }

    if (status >= 300 && status < 400) {
      return explain('redirect', [ref('failure.redirect.steps')]);
    }

    if (status === 429) {
      return explain('rateLimited', [ref('failure.rateLimited.steps')], {
        what,
      });
    }

    if (status >= 500) {
      return explain(
        'serverError',
        [logsStep, ref('failure.serverError.step')],
        { what, status },
      );
    }

    if (status >= 400) {
      return explain('clientError', [logsStep], { what, status });
    }
  }

  if (
    failure.stage === V1ServerlessEndpointCheckStage.PARSE ||
    lower.includes('could not parse healthcheck response')
  ) {
    return explain('notHealthcheck', [
      platform.healthcheckPath
        ? ref('failure.notHealthcheck.step.path', {
            path: platform.healthcheckPath,
          })
        : ref('failure.notHealthcheck.step.noPath'),
      ref('failure.notHealthcheck.step.handler'),
    ]);
  }

  if (
    failure.stage === V1ServerlessEndpointCheckStage.CATALOG ||
    includesAny(lower, [
      'invalid action',
      'advertises',
      'could not register',
      'workflow',
      'task options',
    ])
  ) {
    return explain('catalog', [ref('failure.catalog.steps')]);
  }

  if (lower.includes('exceeded') && lower.includes('limit')) {
    return explain('tooLarge', [ref('failure.tooLarge.steps')], { what });
  }

  if (lower.includes('non-json body')) {
    return explain('nonJson', [logsStep]);
  }

  return {
    title: ref('failure.unknown.title', { what }),
    detail: ref(
      failure.error ? 'failure.unknown.detail' : 'failure.unknown.noMessage',
    ),
    steps: [logsStep],
  };
}

function latestOf(
  events: V1ServerlessEndpointEvent[],
  kinds: V1ServerlessEndpointEventKind[],
): V1ServerlessEndpointEvent | undefined {
  return events
    .filter((e) => kinds.includes(e.kind))
    .reduce<V1ServerlessEndpointEvent | undefined>(
      (latest, e) =>
        !latest || Date.parse(e.createdAt) > Date.parse(latest.createdAt)
          ? e
          : latest,
      undefined,
    );
}

function withSeverity(
  severity: DiagnosisSeverity,
  explanation: FailureExplanation,
): Diagnosis {
  return { ...explanation, severity, detail: [explanation.detail] };
}

const simple = (
  severity: DiagnosisSeverity,
  id: 'disabled' | 'noActions',
): Diagnosis => ({
  severity,
  title: ref(`diagnosis.${id}.title`),
  detail: [ref(`diagnosis.${id}.detail`)],
  steps: [ref(`diagnosis.${id}.steps`)],
});

/**
 * Explains the state of an endpoint in plain language: whether it works, and
 * if not, the most likely reason and what to do.
 */
export function diagnoseEndpoint({
  endpoint,
  events = [],
  deployment,
  now = Date.now(),
}: {
  endpoint: V1ServerlessEndpoint;
  events?: V1ServerlessEndpointEvent[];
  deployment: Deployment;
  now?: number;
}): Diagnosis {
  const { status, kind } = endpoint;

  if (!endpoint.enabled) {
    return simple('info', 'disabled');
  }

  const lastFailure = latestOf(events, [
    V1ServerlessEndpointEventKind.HEALTHCHECK_FAILED,
    V1ServerlessEndpointEventKind.CATALOG_REJECTED,
  ]);
  const lastHealthy = latestOf(events, [
    V1ServerlessEndpointEventKind.HEALTHY,
    V1ServerlessEndpointEventKind.CATALOG_REGISTERED,
  ]);
  const failingSinceHealthy =
    lastFailure &&
    (!lastHealthy ||
      Date.parse(lastFailure.createdAt) > Date.parse(lastHealthy.createdAt));

  if (status.healthy === undefined) {
    if (lastFailure) {
      return withSeverity(
        'error',
        explainFailure(kind, {
          deployment,
          request: 'healthcheck',
          error: lastFailure.message,
          statusCode: lastFailure.details.statusCode,
          stage: lastFailure.details.stage,
        }),
      );
    }

    const pollMs = endpoint.pollIntervalSeconds * 1000;
    const age = now - Date.parse(endpoint.metadata.createdAt);

    // a few poll intervals without a single healthcheck: nothing is polling the endpoint
    if (age > pollMs * 3 + 30_000) {
      return {
        severity: 'warning',
        title: ref('diagnosis.notPolled.title'),
        detail: [ref('diagnosis.notPolled.detail')],
        steps: [ref(`diagnosis.notPolled.steps.${deployment}`)],
      };
    }

    return {
      severity: 'pending',
      title: ref('diagnosis.pending.title'),
      detail: [
        ref('diagnosis.pending.detail', {
          seconds: endpoint.pollIntervalSeconds,
        }),
      ],
      steps: [],
    };
  }

  if (status.healthy === false) {
    return withSeverity(
      'error',
      explainFailure(kind, {
        deployment,
        request: 'healthcheck',
        error: status.error ?? lastFailure?.message,
        statusCode: lastFailure?.details.statusCode,
        stage: lastFailure?.details.stage,
      }),
    );
  }

  const lastDelivery = latestOf(events, [
    V1ServerlessEndpointEventKind.DELIVERY_FAILED,
  ]);

  if (
    lastDelivery &&
    now - Date.parse(lastDelivery.createdAt) < RECENT_DELIVERY_FAILURE_MS
  ) {
    const explanation = explainFailure(kind, {
      deployment,
      request: 'delivery',
      error: lastDelivery.message,
      statusCode: lastDelivery.details.statusCode,
    });

    return {
      severity: 'warning',
      title: ref('diagnosis.deliveriesFailing.title'),
      cause: explanation.title,
      detail: [ref('diagnosis.deliveriesFailing.detail'), explanation.detail],
      steps: explanation.steps,
    };
  }

  if (failingSinceHealthy && lastFailure) {
    const explanation = explainFailure(kind, {
      deployment,
      request: 'healthcheck',
      error: lastFailure.message,
      statusCode: lastFailure.details.statusCode,
      stage: lastFailure.details.stage,
    });

    return {
      severity: 'warning',
      title: ref('diagnosis.healthchecksFailing.title'),
      cause: explanation.title,
      detail: [ref('diagnosis.healthchecksFailing.detail'), explanation.detail],
      steps: explanation.steps,
    };
  }

  if (status.registeredActions.length === 0) {
    return simple('warning', 'noActions');
  }

  const count = status.registeredActions.length;

  return {
    severity: 'healthy',
    title: ref('diagnosis.healthy.title'),
    detail: [
      count === 1
        ? ref('diagnosis.healthy.detailOne')
        : ref('diagnosis.healthy.detail', { count }),
    ],
    steps: [],
  };
}
