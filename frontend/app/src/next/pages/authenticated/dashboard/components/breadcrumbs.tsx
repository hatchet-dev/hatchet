import { useLocation } from 'react-router-dom';
import React, { useMemo } from 'react';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/next/components/ui/breadcrumb';
import {
  getMainNavLinks,
  NavItem as MainNavItem,
  NavSection,
} from '@/next/pages/authenticated/dashboard/components/sidebar/main-nav';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/next/components/ui/dropdown-menu';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { useIsMobile } from '@/next/hooks/use-mobile';
import { BreadcrumbData, useBreadcrumbs } from '@/next/hooks/use-breadcrumbs';

// Use the existing NavItem type from main-nav
type NavItem = MainNavItem;

export function BreadcrumbNav() {
  const location = useLocation();
  const isMobile = useIsMobile();
  const navStructure = getMainNavLinks(location.pathname);

  const { breadcrumbs } = useBreadcrumbs();

  // Flattened navigation map for easy lookup
  const navMap = useMemo(() => new Map<string, NavItem>(), []);

  // Map to track siblings at each level of the hierarchy
  const siblingsByPath = useMemo(() => new Map<string, NavItem[]>(), []);

  // Store section items by their first segment for root level organization
  const sectionItemsByRootPath = useMemo(
    () => new Map<string, NavSection>(),
    [],
  );

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
  const breadcrumbItemsFromNav: BreadcrumbData[] = useMemo(() => {
    const breadcrumbItemsFromNav: BreadcrumbData[] = [];
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

        breadcrumbItemsFromNav.push({
          title: navItem.title,
          url: navItem.url,
          siblings: siblings.length > 1 ? siblings : undefined,
          section: sectionName,
          icon: navItem.icon,
        });
      }
    }
    return breadcrumbItemsFromNav;
  }, [navMap, pathSegments, sectionItemsByRootPath, siblingsByPath]);

  const breadcrumbItems = useMemo<
    (BreadcrumbData & { isLast: boolean; isFirst: boolean })[]
  >(() => {
    const mergedBreadcrumbs = [...breadcrumbItemsFromNav, ...breadcrumbs];

    return mergedBreadcrumbs.map((item, index) => ({
      ...item,
      alwaysShowTitle: item.alwaysShowTitle ?? true,
      alwaysShowIcon: item.alwaysShowIcon ?? true,
      isLast: index === mergedBreadcrumbs.length - 1,
      isFirst: index === 0,
    }));
  }, [breadcrumbItemsFromNav, breadcrumbs]);

  return (
    <Breadcrumb>
      <BreadcrumbList className="flex w-full items-center">
        {breadcrumbItems.map((item, index) => (
          <BreadcrumbItem
            key={item.url + index}
            className={`flex-shrink overflow-hidden ${item.isLast ? 'flex-1' : 'max-w-fit'}`}
          >
            {item.isLast ? (
              item.siblings ? (
                <DropdownMenu>
                  <DropdownMenuTrigger className="flex items-center gap-2 font-normal text-foreground whitespace-nowrap overflow-hidden text-ellipsis">
                    {(item.isFirst || item.alwaysShowIcon) && item.icon && (
                      <item.icon className="h-4 w-4 flex-shrink-0" />
                    )}
                    <span className="overflow-hidden text-ellipsis">
                      {item.title}
                    </span>
                    <ChevronDown className="h-4 w-4 flex-shrink-0 ml-1" />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    {item.siblings.map((sibling, index) => (
                      <DropdownMenuItem key={sibling.url + index} asChild>
                        <BreadcrumbLink
                          to={sibling.url}
                          className="flex items-center gap-2"
                        >
                          {sibling.icon && (
                            <sibling.icon className="h-4 w-4 flex-shrink-0" />
                          )}
                          {sibling.title}
                        </BreadcrumbLink>
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuContent>
                </DropdownMenu>
              ) : (
                <BreadcrumbPage className="whitespace-nowrap overflow-hidden text-ellipsis inline-flex items-center gap-2">
                  {(item.isFirst || item.alwaysShowIcon) && item.icon && (
                    <item.icon className="h-4 w-4 flex-shrink-0" />
                  )}
                  {(item.alwaysShowTitle || !(item.isFirst || isMobile)) && (
                    <span className="overflow-hidden text-ellipsis">
                      {item.title}
                    </span>
                  )}
                </BreadcrumbPage>
              )
            ) : item.siblings ? (
              <div className="group flex items-center">
                <BreadcrumbLink
                  to={item.url}
                  className="flex items-center gap-2 whitespace-nowrap overflow-hidden text-ellipsis"
                >
                  {(item.isFirst || item.alwaysShowIcon) && item.icon && (
                    <item.icon className="h-4 w-4 flex-shrink-0" />
                  )}
                  {(item.alwaysShowTitle || !(item.isFirst || isMobile)) && (
                    <span className="overflow-hidden text-ellipsis">
                      {item.title}
                    </span>
                  )}
                </BreadcrumbLink>
                <div className="relative w-4 h-4 mx-2 flex items-center justify-center">
                  <ChevronRight className="absolute h-4 w-4 group-hover:opacity-0 transition-opacity" />
                  <DropdownMenu>
                    <DropdownMenuTrigger className="absolute inset-0 flex items-center justify-center">
                      <ChevronDown className="h-4 w-4 opacity-0 group-hover:opacity-100 transition-opacity" />
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="start">
                      {item.siblings.map((sibling, index) => (
                        <DropdownMenuItem key={sibling.url + index} asChild>
                          <BreadcrumbLink
                            to={sibling.url}
                            className="flex items-center gap-2"
                          >
                            {sibling.icon && (
                              <sibling.icon className="h-4 w-4 flex-shrink-0" />
                            )}
                            {sibling.title}
                          </BreadcrumbLink>
                        </DropdownMenuItem>
                      ))}
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </div>
            ) : (
              <BreadcrumbLink
                to={item.url}
                className="whitespace-nowrap overflow-hidden text-ellipsis inline-flex items-center gap-2"
              >
                {(item.isFirst || item.alwaysShowIcon) && item.icon && (
                  <item.icon className="h-4 w-4 flex-shrink-0" />
                )}
                {(item.alwaysShowTitle || !(item.isFirst || isMobile)) && (
                  <span className="overflow-hidden text-ellipsis">
                    {item.title}
                  </span>
                )}
              </BreadcrumbLink>
            )}
            {!item.isLast && !item.siblings && <BreadcrumbSeparator />}
          </BreadcrumbItem>
        ))}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
