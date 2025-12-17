import { generateBreadcrumbs, BreadcrumbItem } from '@/lib/breadcrumbs';
import { useLocation, useParams } from '@tanstack/react-router';

export function useBreadcrumbs(): BreadcrumbItem[] {
  const location = useLocation();
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

  return generateBreadcrumbs(location.pathname, cleanParams);
}
