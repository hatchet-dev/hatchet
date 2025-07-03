import { useLocation, useParams } from 'react-router-dom';
import { generateBreadcrumbs, BreadcrumbItem } from '@/lib/breadcrumbs';

export function useBreadcrumbs(): BreadcrumbItem[] {
  const location = useLocation();
  const params = useParams();

  const cleanParams = Object.entries(params).reduce((acc, [key, value]) => {
    if (value !== undefined) {
      acc[key] = value;
    }

    return acc;
  }, {} as Record<string, string>);

  return generateBreadcrumbs(location.pathname, cleanParams);
}
