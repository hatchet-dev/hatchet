type TenantedPath =
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
  | '/tenants/:tenant/settings/overview'
  | '/tenants/:tenant/settings/resource-limits'
  | '/tenants/:tenant/settings/api-tokens'
  | '/tenants/:tenant/settings/github'
  | '/tenants/:tenant/settings/members'
  | '/tenants/:tenant/settings/alerting'
  | '/tenants/:tenant/settings/billing-and-limits'
  | '/tenants/:tenant/settings/ingestors'
  | '/tenants/:tenant/settings/integrations'
  | '/tenants/:tenant/settings/organization'
  | '/tenants/:tenant/workflow-runs'
  | '/tenants/:tenant/workflow-runs/:run'
  | '/tenants/:tenant/'
  | '/tenants/:tenant/workflows'
  | '/tenants/:tenant/workflows/:workflow'
  | '/tenants/:tenant/settings';

type OrganizationPath =
  | '/organizations/:organization/settings'
  | '/organizations/:organization/settings/team'
  | '/organizations/:organization/settings/tenants'
  | '/organizations/:organization/settings/tokens'
  | '/organizations/:organization/settings/regions'
  | '/organizations/:organization/settings/sso'
  | '/organizations/:organization/settings/audit-log'
  | '/organizations/:organization/settings/billing';

export interface BreadcrumbItem {
  label: string;
  href?: string;
  isCurrentPage?: boolean;
}

const createRouteLabel = (
  path: TenantedPath,
  isCloudEnabled: boolean,
): string => {
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
    case '/tenants/:tenant/settings/overview':
      return 'General';
    case '/tenants/:tenant/settings/resource-limits':
      return 'Resource Limits';
    case '/tenants/:tenant/settings/api-tokens':
      return 'API Tokens';
    case '/tenants/:tenant/settings/github':
      return 'GitHub';
    case '/tenants/:tenant/settings/members':
      return 'Members';
    case '/tenants/:tenant/settings/alerting':
      return 'Alerting';
    case '/tenants/:tenant/settings/billing-and-limits':
      return 'Billing & Limits';
    case '/tenants/:tenant/settings/ingestors':
      return 'Ingestors';
    case '/tenants/:tenant/settings/integrations':
      return 'Integrations';
    case '/tenants/:tenant/settings/organization':
      return isCloudEnabled ? 'Organization' : 'Tenants';
    case '/tenants/:tenant/workflow-runs':
    case '/tenants/:tenant/workflow-runs/:run':
    case '/tenants/:tenant/':
    case '/tenants/:tenant/workflows':
    case '/tenants/:tenant/workflows/:workflow':
    case '/tenants/:tenant/settings':
      return '';
    default:
      // eslint-disable-next-line no-case-declarations
      const exhaustiveCheck: never = path;
      throw new Error(`Unhandled tenanted path: ${exhaustiveCheck}`);
  }
};

const createOrganizationRouteLabel = (path: OrganizationPath): string => {
  switch (path) {
    // The organization settings index renders the General section, so it gets
    // the same leaf label as the tenant `/settings/overview` page.
    case '/organizations/:organization/settings':
      return 'General';
    case '/organizations/:organization/settings/team':
      return 'Team';
    case '/organizations/:organization/settings/tenants':
      return 'Tenants';
    case '/organizations/:organization/settings/tokens':
      return 'Management Tokens';
    case '/organizations/:organization/settings/regions':
      return 'Available Regions';
    case '/organizations/:organization/settings/sso':
      return 'SSO';
    case '/organizations/:organization/settings/audit-log':
      return 'Audit Log';
    case '/organizations/:organization/settings/billing':
      return 'Billing & Usage';
    default:
      // eslint-disable-next-line no-case-declarations
      const exhaustiveCheck: never = path;
      throw new Error(`Unhandled organization path: ${exhaustiveCheck}`);
  }
};

function generateOrganizationBreadcrumbs(
  pathname: string,
  organizationId: string,
): BreadcrumbItem[] {
  const normalizedPath = pathname
    .replace(/\/+$/, '')
    .replace(
      `/organizations/${organizationId}`,
      '/organizations/:organization',
    );

  let label: string;
  try {
    label = createOrganizationRouteLabel(normalizedPath as OrganizationPath);
  } catch {
    // Unknown organization paths (e.g. redirect-only routes) get no crumbs.
    return [];
  }

  return [
    {
      // The Home href is patched in `useBreadcrumbs` once the active tenant
      // is resolved (there is no tenant URL param on organization routes).
      label: 'Home',
    },
    {
      label: 'Settings',
      href: `/organizations/${organizationId}/settings`,
    },
    {
      label,
      isCurrentPage: true,
    },
  ];
}

function getTenantedPathLabel(
  pathSegments: string[],
  tenantId: string,
  isCloudEnabled: boolean,
): string | null {
  const fullPath = '/' + pathSegments.join('/');
  const normalizedPath = fullPath.replace(
    `/tenants/${tenantId}`,
    '/tenants/:tenant',
  );

  try {
    const label = createRouteLabel(
      normalizedPath as TenantedPath,
      isCloudEnabled,
    );
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
  isCloudEnabled?: boolean,
): BreadcrumbItem[] {
  const breadcrumbs: BreadcrumbItem[] = [];

  if (pathname.includes('/auth') || pathname.includes('/onboarding')) {
    return breadcrumbs;
  }

  if (pathname.startsWith('/organizations/')) {
    const organizationId = params?.organization;
    if (!organizationId) {
      return breadcrumbs;
    }

    return generateOrganizationBreadcrumbs(pathname, organizationId);
  }

  if (!pathname.startsWith('/tenants/')) {
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

    const label = getTenantedPathLabel(
      segments.slice(0, 2 + i + 1),
      tenantId,
      isCloudEnabled ?? false,
    );

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
