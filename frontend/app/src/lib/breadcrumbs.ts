import { TenantedPath } from "@/router";

export interface BreadcrumbItem {
  label: string;
  href?: string;
  isCurrentPage?: boolean;
}

const createRouteLabel = (path: TenantedPath): string => {
    switch (path) {
        case '/tenants/:tenant/events':
            return 'Events';
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
        case '/tenants/:tenant/workers/webhook':
            return 'Webhook Workers';
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
        case '/tenants/:tenant/onboarding/get-started':
            return 'Get Started';
        case '/tenants/:tenant/workflow-runs':
        case '/tenants/:tenant/workflow-runs/:run':
        case '/tenants/:tenant/':
        case '/tenants/:tenant/workflows':
        case '/tenants/:tenant/workflows/:workflow':
            return '';
        default:
            const exhaustiveCheck: never = path;
            throw new Error(`Unhandled tenanted path: ${exhaustiveCheck}`);
    }
}

function getTenantedPathLabel(pathSegments: string[], tenantId: string): string | null {
  const fullPath = '/' + pathSegments.join('/');
  const normalizedPath = fullPath.replace(`/tenants/${tenantId}`, '/tenants/:tenant');

  try {
    const label = createRouteLabel(normalizedPath as TenantedPath);
    return label || null;
  } catch {
    return null;
  }
}

// This is a hack that relies way too much on pattern matching
// We should replace this later with something more robust
function getDynamicLabel(params?: Record<string, string>): string | null {
  if (params?.workflow) {
    return `Task: ${decodeURIComponent(params.workflow)}`;
  }

  if (params?.run) {
    return `Run: ${params.run}`;
  }

  if (params?.worker) {
    return `Worker: ${decodeURIComponent(params.worker)}`;
  }

  if (params?.['managed-worker']) {
    return `Managed Worker: ${decodeURIComponent(params['managed-worker'])}`;
  }

  return null;
}

function buildParentPath(segments: string[], index: number, tenantId: string): string {
  const tenantPrefix = `/tenants/${tenantId}`;
  if (index === 0) return tenantPrefix;
  return tenantPrefix + '/' + segments.slice(1, index + 1).join('/');
}

export function generateBreadcrumbs(pathname: string, params?: Record<string, string>): BreadcrumbItem[] {
  const breadcrumbs: BreadcrumbItem[] = [];

  if (!pathname.startsWith('/tenants/') || pathname.includes('/auth') || pathname.includes('/onboarding')) {
    return breadcrumbs;
  }

  const tenantId = params?.tenant;
  if (!tenantId) return breadcrumbs;

  breadcrumbs.push({
    label: 'Home',
    href: `/tenants/${tenantId}/runs`,
  });

  const segments = pathname.split('/').filter(Boolean);
  const relevantSegments = segments.slice(2);

  for (let i = 0; i < relevantSegments.length; i++) {
    const currentPath = buildParentPath(segments, 2 + i, tenantId);
    const isLastSegment = i === relevantSegments.length - 1;

    const dynamicLabel = getDynamicLabel(params);
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
        .map(word => word.charAt(0).toUpperCase() + word.slice(1))
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


