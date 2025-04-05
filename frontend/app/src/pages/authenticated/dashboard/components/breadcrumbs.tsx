import { useLocation } from 'react-router-dom';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';
import { getMainNavLinks } from '@/pages/authenticated/dashboard/components/sidebar/main-nav';

type NavItem = {
  title: string;
  url: string;
  items?: NavItem[];
};

export function BreadcrumbNav() {
  const location = useLocation();
  const navStructure = getMainNavLinks(location.pathname);

  // If we're at the root, don't show breadcrumbs
  if (location.pathname === '/') {
    return null;
  }

  // Flattened navigation map for easy lookup
  const navMap = new Map<string, NavItem>();

  // Helper function to add items to the map
  const addToMap = (items: NavItem[]) => {
    items.forEach((item) => {
      navMap.set(item.url, item);
      if (item.items) {
        addToMap(item.items);
      }
    });
  };

  // Process all sections and flatten the structure
  Object.values(navStructure.sections).forEach((section) => {
    addToMap(section.items);
  });

  // Build breadcrumb path based on current location
  const pathSegments = location.pathname.split('/').filter(Boolean);
  const breadcrumbItems: { title: string; url: string; isLast: boolean }[] = [];

  // Build path segments and find matching nav items
  let currentPath = '';
  for (let i = 0; i < pathSegments.length; i++) {
    currentPath += '/' + pathSegments[i];
    const navItem = navMap.get(currentPath);

    if (navItem) {
      breadcrumbItems.push({
        title: navItem.title,
        url: navItem.url,
        isLast: i === pathSegments.length - 1,
      });
    }
  }

  return (
    <Breadcrumb>
      <BreadcrumbList>
        {breadcrumbItems.map((item, index) => (
          <BreadcrumbItem key={item.url}>
            {item.isLast ? (
              <BreadcrumbPage>{item.title}</BreadcrumbPage>
            ) : (
              <BreadcrumbLink to={item.url}>{item.title}</BreadcrumbLink>
            )}
            {!item.isLast && <BreadcrumbSeparator />}
          </BreadcrumbItem>
        ))}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
