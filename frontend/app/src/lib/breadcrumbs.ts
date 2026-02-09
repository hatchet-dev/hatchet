export type TenantedPath =
  | '/tenants/:tenant/events'
  | '/tenants/:tenant/filters'
  | '/tenants/:tenant/webhooks'
  | '/tenants/:tenant/rate-limits'
  | '/tenants/:tenant/scheduled'
  | '/tenants/:tenant/cron-jobs'
  | '/tenants/:tenant/tasks'
  | '/tenants/:tenant/tasks/:workflow'
  | '/tenants/:tenant/runs'
  | '/tenants/:tenant/runs/:run'
  | '/tenants/:tenant/task-runs/:run'
  | '/tenants/:tenant/workers'
  | '/tenants/:tenant/workers/all'
  | '/tenants/:tenant/workers/:worker'
  | '/tenants/:tenant/managed-workers'
  | '/tenants/:tenant/managed-workers/create'
  | '/tenants/:tenant/managed-workers/demo-template'
  | '/tenants/:tenant/managed-workers/:managed-worker'
  | '/tenants/:tenant/tenant-settings/overview'
  | '/tenants/:tenant/tenant-settings/api-tokens'
  | '/tenants/:tenant/tenant-settings/github'
  | '/tenants/:tenant/tenant-settings/members'
  | '/tenants/:tenant/tenant-settings/alerting'
  | '/tenants/:tenant/tenant-settings/billing-and-limits'
  | '/tenants/:tenant/tenant-settings/ingestors'
  | '/tenants/:tenant/workflow-runs'
  | '/tenants/:tenant/workflow-runs/:run'
  | '/tenants/:tenant/'
  | '/tenants/:tenant/workflows'
  | '/tenants/:tenant/workflows/:workflow'
  | '/tenants/:tenant/tenant-settings';

export interface BreadcrumbItem {
  label: string;
  href?: string;
  isCurrentPage?: boolean;
}

const createRouteLabel = (path: TenantedPath): string => {
  switch (path) {
    case '/tenants/:tenant/events':
      return 'Events';
    case '/tenants/:tenant/filters':
      return 'Filters';
    case '/tenants/:tenant/webhooks':
      return 'Webhooks';
    case '/tenants/:tenant/rate-limits':
      return 'Rate Limits';
    case '/tenants/:tenant/scheduled':
      return 'Scheduled Runs';
    case '/tenants/:tenant/cron-jobs':
      return 'Cron Jobs';
    case '/tenants/:tenant/tasks':
      return 'Tasks';
    case '/tenants/:tenant/tasks/:workflow':
      return 'Task Detail';
    case '/tenants/:tenant/runs':
      return 'Runs';
    case '/tenants/:tenant/runs/:run':
      return 'Run Detail';
    case '/tenants/:tenant/task-runs/:run':
      return 'Task Run Detail';
    case '/tenants/:tenant/workers':
      return 'Workers';
    case '/tenants/:tenant/workers/all':
      return 'All Workers';
    case '/tenants/:tenant/workers/:worker':
      return 'Worker Detail';
    case '/tenants/:tenant/managed-workers':
      return 'Managed Workers';
    case '/tenants/:tenant/managed-workers/create':
      return 'Create';
    case '/tenants/:tenant/managed-workers/demo-template':
      return 'Demo Template';
    case '/tenants/:tenant/managed-workers/:managed-worker':
      return 'Managed Worker Detail';
    case '/tenants/:tenant/tenant-settings/overview':
      return 'Overview';
    case '/tenants/:tenant/tenant-settings/api-tokens':
      return 'API Tokens';
    case '/tenants/:tenant/tenant-settings/github':
      return 'GitHub';
    case '/tenants/:tenant/tenant-settings/members':
      return 'Members';
    case '/tenants/:tenant/tenant-settings/alerting':
      return 'Alerting';
    case '/tenants/:tenant/tenant-settings/billing-and-limits':
      return 'Billing & Limits';
    case '/tenants/:tenant/tenant-settings/ingestors':
      return 'Ingestors';
    case '/tenants/:tenant/workflow-runs':
    case '/tenants/:tenant/workflow-runs/:run':
    case '/tenants/:tenant/':
    case '/tenants/:tenant/workflows':
    case '/tenants/:tenant/workflows/:workflow':
    case '/tenants/:tenant/tenant-settings':
      return '';
    default:
      // eslint-disable-next-line no-case-declarations
      const exhaustiveCheck: never = path;
      throw new Error(`Unhandled tenanted path: ${exhaustiveCheck}`);
  }
};

function getTenantedPathLabel(
  pathSegments: string[],
  tenantId: string,
): string | null {
  const fullPath = '/' + pathSegments.join('/');
  const normalizedPath = fullPath.replace(
    `/tenants/${tenantId}`,
    '/tenants/:tenant',
  );

  try {
    const label = createRouteLabel(normalizedPath as TenantedPath);
    return label || null;
  } catch {
    return null;
  }
}

function buildParentPath(
  segments: string[],
  index: number,
  tenantId: string,
): string {
  const tenantPrefix = `/tenants/${tenantId}`;
  if (index === 0) {
    return tenantPrefix;
  }
  return tenantPrefix + '/' + segments.slice(2, index + 1).join('/');
}

export function generateBreadcrumbs(
  pathname: string,
  params?: Record<string, string>,
): BreadcrumbItem[] {
  const breadcrumbs: BreadcrumbItem[] = [];

  if (
    !pathname.startsWith('/tenants/') ||
    pathname.includes('/auth') ||
    pathname.includes('/onboarding')
  ) {
    return breadcrumbs;
  }

  const tenantId = params?.tenant;
  if (!tenantId) {
    return breadcrumbs;
  }

  breadcrumbs.push({
    label: 'Home',
    href: `/tenants/${tenantId}/runs`,
  });

  const segments = pathname.split('/').filter(Boolean);
  const relevantSegments = segments.slice(2);

  for (let i = 0; i < relevantSegments.length; i++) {
    const currentPath = buildParentPath(segments, 2 + i, tenantId);
    const isLastSegment = i === relevantSegments.length - 1;
    const currentSegment = relevantSegments[i];

    // Check if this specific segment is a dynamic parameter
    let dynamicLabel: string | null = null;
    if (params?.workflow && currentSegment === params.workflow) {
      dynamicLabel = params.workflow;
    } else if (params?.run && currentSegment === params.run) {
      dynamicLabel = params.run;
    } else if (params?.worker && currentSegment === params.worker) {
      dynamicLabel = params.worker;
    } else if (
      params?.['managed-worker'] &&
      currentSegment === params['managed-worker']
    ) {
      dynamicLabel = params['managed-worker'];
    }

    if (dynamicLabel) {
      breadcrumbs.push({
        label: dynamicLabel,
        href: isLastSegment ? undefined : currentPath,
        isCurrentPage: isLastSegment,
      });
      continue;
    }

    const label = getTenantedPathLabel(segments.slice(0, 2 + i + 1), tenantId);

    if (label) {
      breadcrumbs.push({
        label,
        href: isLastSegment ? undefined : currentPath,
        isCurrentPage: isLastSegment,
      });
    } else {
      const fallbackLabel = relevantSegments[i]
        .split('-')
        .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
        .join(' ');

      breadcrumbs.push({
        label: fallbackLabel,
        href: isLastSegment ? undefined : currentPath,
        isCurrentPage: isLastSegment,
      });
    }
  }

  return breadcrumbs;
}
