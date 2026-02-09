import { useTenantHomeRoute } from '@/hooks/use-tenant-home-route';
import { generateBreadcrumbs, BreadcrumbItem } from '@/lib/breadcrumbs';
import { useLocation, useParams, useRouterState } from '@tanstack/react-router';

export function useBreadcrumbs(): BreadcrumbItem[] {
  const location = useLocation();
  const routerState = useRouterState();
  // Use non-strict params so breadcrumb generation works outside tenant routes
  // (e.g., onboarding/auth pages) without throwing.
  const params = useParams({ strict: false });

  const cleanParams = Object.entries(params).reduce(
    (acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = value;
      }

      return acc;
    },
    {} as Record<string, string>,
  );

  // Get tenant ID and home route at the top level (before any conditional returns)
  const tenantId = cleanParams.tenant;
  const { homeRoute } = useTenantHomeRoute(tenantId);

  // Check if we're on a tenant route but no child route matched
  // When on a 404 page under /tenants/:tenant/*, the matches will only include
  // the tenant route itself, not any child routes
  const isOnTenantRoute = routerState.matches.some(
    (match) => match.routeId === '/tenants/$tenant',
  );

  const pathSegments = location.pathname.split('/').filter(Boolean);
  const hasChildPath = pathSegments.length > 2; // More than just /tenants/:tenant

  // If we're on a tenant route with a child path but only 3 matches (root, authenticated, tenant),
  // it means the child route didn't match and we're showing notFoundComponent
  const isNotFound =
    isOnTenantRoute && hasChildPath && routerState.matches.length === 3;

  // If we're on a 404 page, return empty breadcrumbs
  if (isNotFound) {
    return [];
  }

  const breadcrumbs = generateBreadcrumbs(location.pathname, cleanParams);

  // Update the Home breadcrumb href with the conditional route
  if (breadcrumbs.length > 0 && breadcrumbs[0].label === 'Home' && tenantId) {
    breadcrumbs[0].href = homeRoute.replace(':tenant', tenantId);
  }

  return breadcrumbs;
}
