import { useLocation } from 'react-router-dom';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';
import {
  getMainNavLinks,
  NavItem as MainNavItem,
  NavSection,
} from '@/pages/authenticated/dashboard/components/sidebar/main-nav';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { ChevronDown } from 'lucide-react';

// Use the existing NavItem type from main-nav
type NavItem = MainNavItem;

export function BreadcrumbNav() {
  const location = useLocation();
  const navStructure = getMainNavLinks(location.pathname);

  // If we're at the root, don't show breadcrumbs
  if (location.pathname === '/') {
    return null;
  }

  // Flattened navigation map for easy lookup
  const navMap = new Map<string, NavItem>();

  // Map to track siblings at each level of the hierarchy
  const siblingsByPath = new Map<string, NavItem[]>();

  // Store section items by their first segment for root level organization
  const sectionItemsByRootPath = new Map<string, NavSection>();

  // Process all sections and organize them
  Object.values(navStructure.sections).forEach((section) => {
    section.items.forEach((item) => {
      // Get the first segment of the URL path (e.g., '/runs' -> 'runs')
      const rootSegment = item.url.split('/').filter(Boolean)[0];
      if (rootSegment) {
        sectionItemsByRootPath.set(rootSegment, section);
      }
    });
  });

  // Helper function to add items to the map
  const addToMap = (items: NavItem[], parentPath = '') => {
    const siblings: NavItem[] = [];

    items.forEach((item) => {
      navMap.set(item.url, item);
      siblings.push(item);

      if (item.items) {
        addToMap(item.items, item.url);
      }
    });

    if (parentPath) {
      siblingsByPath.set(parentPath, siblings);
    }
  };

  // First, collect all root-level items across all sections
  const rootItems: NavItem[] = [];
  Object.values(navStructure.sections).forEach((section) => {
    rootItems.push(...section.items);
  });

  // Set the root siblings
  siblingsByPath.set('/', rootItems);

  // Now process all items in each section
  Object.values(navStructure.sections).forEach((section) => {
    addToMap(section.items);
  });

  // Build breadcrumb path based on current location
  const pathSegments = location.pathname.split('/').filter(Boolean);
  const breadcrumbItems: {
    title: string;
    url: string;
    isLast: boolean;
    siblings?: NavItem[];
    section?: string;
  }[] = [];

  // Build path segments and find matching nav items
  let currentPath = '';
  for (let i = 0; i < pathSegments.length; i++) {
    currentPath += '/' + pathSegments[i];
    const navItem = navMap.get(currentPath);

    if (navItem) {
      // Find siblings for the current item
      const parentPath =
        i === 0 ? '/' : `/${pathSegments.slice(0, i).join('/')}`;
      const siblings = siblingsByPath.get(parentPath) || [];

      // Get section name for first-level items
      let sectionName;
      if (i === 0) {
        const section = sectionItemsByRootPath.get(pathSegments[0]);
        if (section) {
          sectionName = section.label;
        }
      }

      breadcrumbItems.push({
        title: navItem.title,
        url: navItem.url,
        isLast: i === pathSegments.length - 1,
        siblings: siblings.length > 1 ? siblings : undefined,
        section: sectionName,
      });
    }
  }

  return (
    <Breadcrumb>
      <BreadcrumbList>
        {breadcrumbItems.map((item) => (
          <BreadcrumbItem key={item.url}>
            {item.isLast ? (
              item.siblings ? (
                <DropdownMenu>
                  <DropdownMenuTrigger className="flex items-center gap-1 font-normal text-foreground">
                    {item.title}
                    {item.isLast && <ChevronDown className="h-4 w-4" />}
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    {item.siblings.map((sibling) => (
                      <DropdownMenuItem key={sibling.url} asChild>
                        <BreadcrumbLink to={sibling.url}>
                          {sibling.icon && (
                            <sibling.icon className="mr-2 h-4 w-4" />
                          )}
                          {sibling.title}
                        </BreadcrumbLink>
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuContent>
                </DropdownMenu>
              ) : (
                <BreadcrumbPage>{item.title}</BreadcrumbPage>
              )
            ) : item.siblings ? (
              <DropdownMenu>
                <DropdownMenuTrigger className="flex items-center gap-1 transition-colors hover:text-foreground">
                  {item.title}
                  {item.isLast && <ChevronDown className="h-4 w-4" />}
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start">
                  {item.siblings.map((sibling) => (
                    <DropdownMenuItem key={sibling.url} asChild>
                      <BreadcrumbLink to={sibling.url}>
                        {sibling.icon && (
                          <sibling.icon className="mr-2 h-4 w-4" />
                        )}
                        {sibling.title}
                      </BreadcrumbLink>
                    </DropdownMenuItem>
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>
            ) : (
              <BreadcrumbLink to={item.url}>
                {item.title}
                {item.isLast && <ChevronDown className="h-4 w-4" />}
              </BreadcrumbLink>
            )}
            {!item.isLast && <BreadcrumbSeparator />}
          </BreadcrumbItem>
        ))}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
