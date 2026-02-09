import {
  COLLAPSE_SNAP_AT,
  COLLAPSED_SIDEBAR_WIDTH,
  EXPAND_SNAP_AT,
  MAX_EXPANDED_SIDEBAR_WIDTH,
  MIN_EXPANDED_SIDEBAR_WIDTH,
  RESIZE_DRAG_THRESHOLD_PX,
  useSidebar,
} from '@/components/hooks/use-sidebar';
import { HelpDropdown } from '@/components/v1/nav/help-dropdown';
import {
  SidebarButtonPrimary,
  SidebarButtonSecondary,
} from '@/components/v1/nav/sidebar-buttons';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { useTenantDetails } from '@/hooks/use-tenant';
import { cn } from '@/lib/utils';
import { ChevronLeftIcon, ChevronRightIcon } from '@radix-ui/react-icons';
import { Link, useMatchRoute, useNavigate } from '@tanstack/react-router';
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';

export interface SideNavProps extends React.HTMLAttributes<HTMLDivElement> {
  navItems: SideNavSection[];
}

export type SideNavChild = {
  key: string;
  name: string;
  to: string;
};

export type SideNavItem = {
  key: string;
  name: string;
  to: string;
  icon: (opts: { collapsed: boolean; active?: boolean }) => React.ReactNode;
  prefix?: string;
  activeTo?: string;
  activeFuzzy?: boolean;
  children?: SideNavChild[];
};

export type SideNavSection = {
  key: string;
  title: string;
  itemsClassName: string;
  items: SideNavItem[];
};

export function SideNav({ className, navItems: navSections }: SideNavProps) {
  const {
    sidebarOpen,
    setSidebarOpen,
    isWide,
    collapsed: storedCollapsed,
    setCollapsed: setStoredCollapsed,
    expandedWidth: storedExpandedWidth,
    setExpandedWidth: setStoredExpandedWidth,
  } = useSidebar();
  const { tenantId } = useTenantDetails();
  const navigate = useNavigate();
  const matchRoute = useMatchRoute();

  const [isResizing, setIsResizing] = useState(false);
  const [liveWidth, setLiveWidth] = useState<number | null>(null);
  const [startX, setStartX] = useState(0);
  const [startWidth, setStartWidth] = useState(0);
  const wasCollapsedAtDragStartRef = useRef(false);
  const didDragRef = useRef(false);
  const [showResizeToggle, setShowResizeToggle] = useState(false);
  const sidebarRef = useRef<HTMLDivElement>(null);

  const onNavLinkClick = useCallback(() => {
    if (isWide) {
      return;
    }

    setSidebarOpen('closed');
  }, [isWide, setSidebarOpen]);

  const renderCollapsed = (() => {
    if (!isWide) {
      return false;
    }

    if (isResizing && liveWidth !== null) {
      if (wasCollapsedAtDragStartRef.current) {
        return liveWidth < EXPAND_SNAP_AT;
      }

      return liveWidth <= COLLAPSE_SNAP_AT;
    }

    return storedCollapsed;
  })();

  const effectiveWidth = (() => {
    if (!isWide) {
      return undefined;
    }

    if (isResizing && liveWidth !== null) {
      return liveWidth;
    }

    if (storedCollapsed) {
      return COLLAPSED_SIDEBAR_WIDTH;
    }

    return storedExpandedWidth;
  })();

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isResizing) {
        return;
      }

      const deltaX = e.clientX - startX;
      if (!didDragRef.current && Math.abs(deltaX) >= RESIZE_DRAG_THRESHOLD_PX) {
        didDragRef.current = true;
      }
      const newWidth = Math.max(
        COLLAPSED_SIDEBAR_WIDTH,
        Math.min(MAX_EXPANDED_SIDEBAR_WIDTH, startWidth + deltaX),
      );

      setLiveWidth(newWidth);
    },
    [isResizing, startX, startWidth],
  );

  const handleMouseUp = useCallback(() => {
    // If this was a click (no real drag), toggle collapsed state.
    if (!didDragRef.current) {
      if (wasCollapsedAtDragStartRef.current) {
        setStoredCollapsed(false);
      } else {
        setStoredCollapsed(true);
      }

      setLiveWidth(null);
      setIsResizing(false);
      return;
    }

    const finalWidth =
      liveWidth ??
      (storedCollapsed ? COLLAPSED_SIDEBAR_WIDTH : storedExpandedWidth);

    // Snap rules depend on the mode we started dragging in.
    if (wasCollapsedAtDragStartRef.current) {
      if (finalWidth >= EXPAND_SNAP_AT) {
        setStoredCollapsed(false);
        setStoredExpandedWidth(
          Math.max(
            MIN_EXPANDED_SIDEBAR_WIDTH,
            Math.min(MAX_EXPANDED_SIDEBAR_WIDTH, finalWidth),
          ),
        );
      } else {
        setStoredCollapsed(true);
      }
    } else {
      if (finalWidth <= COLLAPSE_SNAP_AT) {
        setStoredCollapsed(true);
      } else {
        setStoredCollapsed(false);
        setStoredExpandedWidth(
          Math.max(
            MIN_EXPANDED_SIDEBAR_WIDTH,
            Math.min(MAX_EXPANDED_SIDEBAR_WIDTH, finalWidth),
          ),
        );
      }
    }

    setLiveWidth(null);
    setIsResizing(false);
  }, [
    liveWidth,
    storedCollapsed,
    storedExpandedWidth,
    setStoredCollapsed,
    setStoredExpandedWidth,
  ]);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      if (!isWide) {
        return;
      }

      e.preventDefault();
      wasCollapsedAtDragStartRef.current = storedCollapsed;
      didDragRef.current = false;
      setIsResizing(true);
      setStartX(e.clientX);
      const start = storedCollapsed
        ? COLLAPSED_SIDEBAR_WIDTH
        : storedExpandedWidth;
      setStartWidth(start);
      setLiveWidth(start);
    },
    [isWide, storedCollapsed, storedExpandedWidth],
  );

  useEffect(() => {
    if (!isResizing) {
      return;
    }

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
    document.body.style.cursor = 'col-resize';

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
    };
  }, [isResizing, handleMouseMove, handleMouseUp]);

  const commonParams = useMemo(
    () => (tenantId ? { tenant: tenantId } : undefined),
    [tenantId],
  );
  const isActive = useCallback(
    (to: string, fuzzy = false) =>
      Boolean(matchRoute({ to, params: commonParams, fuzzy })),
    [matchRoute, commonParams],
  );

  const toggleCollapsed = useCallback(() => {
    setStoredCollapsed(!storedCollapsed);
    setLiveWidth(null);
    setIsResizing(false);
  }, [setStoredCollapsed, storedCollapsed]);

  if (sidebarOpen === 'closed') {
    return null;
  }

  return (
    <div
      ref={sidebarRef}
      data-cy="v1-sidebar"
      className={cn(
        // On mobile, overlay the content area (which is already positioned below the fixed header).
        // On desktop, participate in the grid as a fixed-width sidebar.
        'absolute inset-x-0 top-0 bottom-0 z-[100] w-full overflow-hidden bg-slate-100 dark:bg-slate-900 md:relative md:inset-auto md:top-0 md:bottom-auto md:h-full md:bg-[unset] md:dark:bg-[unset]',
        !isResizing && 'md:transition-[width] md:duration-200 md:ease-in-out',
        className,
      )}
      style={
        isWide
          ? {
              width: effectiveWidth,
              minWidth:
                isResizing || storedCollapsed
                  ? COLLAPSED_SIDEBAR_WIDTH
                  : MIN_EXPANDED_SIDEBAR_WIDTH,
              maxWidth: MAX_EXPANDED_SIDEBAR_WIDTH,
            }
          : undefined
      }
    >
      {/* Desktop-only drag handle */}
      <div
        className={cn(
          'absolute right-0 top-0 bottom-0 z-20 hidden w-1 cursor-col-resize transition-colors hover:bg-blue-500/20 md:block',
          isResizing && 'bg-blue-500/30',
        )}
        data-cy="v1-sidebar-resize-handle"
        onMouseDown={handleMouseDown}
        onMouseUp={(e) => {
          // Handle click-up inside the handle immediately (and prevent the document listener
          // from also firing for the same event).
          e.stopPropagation();
          handleMouseUp();
        }}
        onMouseEnter={() => setShowResizeToggle(true)}
        onMouseLeave={() => setShowResizeToggle(false)}
      >
        {/* Hover affordance: click-to-toggle button on the gutter */}
        {isWide && showResizeToggle && (
          <Button
            variant="ghost"
            size="icon"
            hoverText={storedCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            hoverTextSide="right"
            aria-label={storedCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            className={cn(
              // A small pill that sits inside the gutter (no overflow required)
              'absolute right-0 top-1/2 z-30 hidden h-8 w-5 -translate-y-1/2 rounded-l-md border border-r-0 bg-secondary/90 text-secondary-foreground shadow-sm opacity-0 backdrop-blur transition-opacity md:flex',
              'opacity-100',
              isResizing && 'pointer-events-none opacity-0',
            )}
            data-cy="v1-sidebar-resize-toggle"
            onMouseDown={(e) => {
              // Prevent starting a drag resize when clicking the button.
              e.stopPropagation();
            }}
            onClick={(e) => {
              e.stopPropagation();
              toggleCollapsed();
            }}
          >
            {storedCollapsed ? (
              <ChevronRightIcon className="size-4" />
            ) : (
              <ChevronLeftIcon className="size-4" />
            )}
          </Button>
        )}
      </div>

      <div className="flex h-full flex-col overflow-hidden">
        {renderCollapsed ? (
          <>
            {/* Scrollable navigation area (collapsed) */}
            <div
              data-cy="v1-sidebar-scroll-collapsed"
              className="min-h-0 flex-1 overflow-auto py-4 pl-2 overscroll-contain touch-pan-y [-webkit-overflow-scrolling:touch] [scrollbar-gutter:stable] scrollbar-thin scrollbar-track-transparent scrollbar-thumb-muted-foreground"
            >
              <div className="flex w-full flex-col items-center gap-y-2 px-2">
                {navSections.map((section, sectionIdx) => (
                  <React.Fragment key={section.key}>
                    {sectionIdx > 0 && (
                      <div className="my-2 h-px w-8 bg-slate-200 dark:bg-slate-800" />
                    )}

                    {section.items.map((item) => {
                      const activeTo = item.activeTo ?? item.to;
                      const activeFuzzy = item.activeFuzzy ?? false;
                      const active = isActive(activeTo, activeFuzzy);

                      if (item.children && item.children.length > 0) {
                        return (
                          <DropdownMenu key={item.key}>
                            <DropdownMenuTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon"
                                hoverText={item.name}
                                hoverTextSide="right"
                                aria-label={item.name}
                                className={cn(
                                  'w-10',
                                  active && 'bg-slate-200 dark:bg-slate-800',
                                )}
                              >
                                {item.icon({ collapsed: true, active })}
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent
                              side="right"
                              align="start"
                              className="z-[100]"
                            >
                              {item.children.map((child) => (
                                <DropdownMenuItem
                                  key={child.key}
                                  asChild
                                  className="w-full cursor-pointer"
                                >
                                  <Link
                                    to={child.to}
                                    params={commonParams}
                                    onClick={onNavLinkClick}
                                  >
                                    {child.name}
                                  </Link>
                                </DropdownMenuItem>
                              ))}
                            </DropdownMenuContent>
                          </DropdownMenu>
                        );
                      }

                      return (
                        <Button
                          key={item.key}
                          variant="ghost"
                          size="icon"
                          hoverText={item.name}
                          hoverTextSide="right"
                          aria-label={item.name}
                          className={cn(
                            'w-10',
                            active && 'bg-slate-200 dark:bg-slate-800',
                          )}
                          onClick={() => {
                            navigate({
                              to: item.to,
                              params: commonParams,
                            });
                            onNavLinkClick();
                          }}
                        >
                          {item.icon({ collapsed: true, active })}
                        </Button>
                      );
                    })}
                  </React.Fragment>
                ))}
              </div>
            </div>

            {/* Fixed footer */}
            <div className="w-full shrink-0 py-4">
              <div className="flex w-full justify-center">
                <HelpDropdown
                  variant="sidebar"
                  triggerVariant="icon"
                  align="start"
                  side="right"
                  className="w-10"
                />
              </div>
            </div>
          </>
        ) : (
          <>
            {/* Scrollable navigation area (keep scrollbar flush to sidebar edge) */}
            <div
              data-cy="v1-sidebar-scroll"
              className="min-h-0 flex-1 overflow-auto overscroll-contain touch-pan-y [-webkit-overflow-scrolling:touch] [scrollbar-gutter:stable] scrollbar-thin scrollbar-track-transparent scrollbar-thumb-muted-foreground"
            >
              <div className="px-4 py-4">
                {navSections.map((section) => (
                  <div key={section.key} className="py-2">
                    <h2 className="mb-2 text-xs font-mono tracking-widest uppercase text-muted-foreground">
                      {section.title}
                    </h2>

                    <div className={section.itemsClassName}>
                      {section.items.map((item) => (
                        <SidebarButtonPrimary
                          key={item.key}
                          onNavLinkClick={onNavLinkClick}
                          to={item.to}
                          params={commonParams}
                          prefix={item.prefix}
                          name={item.name}
                          icon={item.icon({
                            collapsed: false,
                            active: isActive(item.to, item.activeFuzzy),
                          })}
                          collapsibleChildren={
                            item.children?.map((child) => (
                              <SidebarButtonSecondary
                                key={child.key}
                                onNavLinkClick={onNavLinkClick}
                                to={child.to}
                                params={commonParams}
                                name={child.name}
                              />
                            )) ?? []
                          }
                        />
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Fixed footer: tenant/org picker is always visible and takes up space */}
            <div
              data-cy="v1-sidebar-footer"
              className="w-full shrink-0 border-t border-slate-200 px-4 py-4 dark:border-slate-800"
            >
              <HelpDropdown
                variant="sidebar"
                triggerVariant="split"
                align="start"
                side="top"
                className="w-full"
              />
            </div>
          </>
        )}
      </div>
    </div>
  );
}
