import { ThreeColumnLayout } from '@/components/layout/three-column-layout';
import { SidePanel } from '@/components/side-panel';
import { useSidebar } from '@/components/sidebar-provider';
import { OrganizationSelector } from '@/components/v1/molecules/nav-bar/organization-selector';
import { TenantSwitcher } from '@/components/v1/molecules/nav-bar/tenant-switcher';
import { Button } from '@/components/v1/ui/button';
import {
  Collapsible,
  CollapsibleContent,
} from '@/components/v1/ui/collapsible';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { Loading } from '@/components/v1/ui/loading.tsx';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { SidePanelProvider } from '@/hooks/use-side-panel';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { TenantMember } from '@/lib/api';
import {
  MembershipsContextType,
  UserContextType,
  useContextFromParent,
} from '@/lib/outlet';
import { OutletWithContext, useOutletContext } from '@/lib/router-helpers';
import { cn } from '@/lib/utils';
import useCloudApiMeta from '@/pages/auth/hooks/use-cloud-api-meta';
import useCloudFeatureFlags from '@/pages/auth/hooks/use-cloud-feature-flags';
import { appRoutes } from '@/router';
import {
  BuildingOffice2Icon,
  CalendarDaysIcon,
  CpuChipIcon,
  PlayIcon,
  ScaleIcon,
  ServerStackIcon,
  Squares2X2Icon,
} from '@heroicons/react/24/outline';
import { ClockIcon, GearIcon } from '@radix-ui/react-icons';
import { Link, useMatchRoute, useNavigate } from '@tanstack/react-router';
import { Filter, SquareActivityIcon, WebhookIcon } from 'lucide-react';
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';

function Main() {
  const ctx = useOutletContext<UserContextType & MembershipsContextType>();
  const { user, memberships } = ctx;

  const childCtx = useContextFromParent({
    user,
    memberships,
  });

  if (!user || !memberships) {
    return <Loading />;
  }

  return (
    <SidePanelProvider>
      <ThreeColumnLayout
        sidebar={<Sidebar memberships={memberships} />}
        sidePanel={<SidePanel />}
        mainClassName="overflow-auto px-8 py-4"
        mainContainerType="inline-size"
      >
        <OutletWithContext context={childCtx} />
      </ThreeColumnLayout>
    </SidePanelProvider>
  );
}

export default Main;

interface SidebarProps extends React.HTMLAttributes<HTMLDivElement> {
  memberships: TenantMember[];
}

const DEFAULT_EXPANDED_SIDEBAR_WIDTH = 200; // matches prior `w-64`
const MIN_EXPANDED_SIDEBAR_WIDTH = 200;
const MAX_EXPANDED_SIDEBAR_WIDTH = 520;
const COLLAPSED_SIDEBAR_WIDTH = 56;
const COLLAPSE_SNAP_AT = MIN_EXPANDED_SIDEBAR_WIDTH;
const EXPAND_SNAP_AT = MIN_EXPANDED_SIDEBAR_WIDTH - 100;
const RESIZE_DRAG_THRESHOLD_PX = 3;

type SidebarNavChild = {
  key: string;
  name: string;
  to: string;
};

type SidebarNavItem = {
  key: string;
  name: string;
  to: string;
  icon: (opts: { collapsed: boolean }) => React.ReactNode;
  prefix?: string;
  activeTo?: string;
  activeFuzzy?: boolean;
  children?: SidebarNavChild[];
};

type SidebarNavSection = {
  key: string;
  title: string;
  itemsClassName: string;
  items: SidebarNavItem[];
};

function Sidebar({ className, memberships }: SidebarProps) {
  const { sidebarOpen, setSidebarOpen } = useSidebar();
  const { tenantId } = useCurrentTenantId();
  const navigate = useNavigate();
  const matchRoute = useMatchRoute();

  const { data: cloudMeta, isCloudEnabled } = useCloudApiMeta();
  const featureFlags = useCloudFeatureFlags(tenantId);

  const defaultExpandedWidth = (() => {
    if (typeof window === 'undefined') {
      return DEFAULT_EXPANDED_SIDEBAR_WIDTH;
    }

    try {
      // Back-compat: previous implementation stored this under `v1SidebarWidth`.
      const legacy = window.localStorage.getItem('v1SidebarWidth');
      return legacy ? JSON.parse(legacy) : DEFAULT_EXPANDED_SIDEBAR_WIDTH;
    } catch {
      return DEFAULT_EXPANDED_SIDEBAR_WIDTH;
    }
  })();

  const [storedExpandedWidth, setStoredExpandedWidth] = useLocalStorageState(
    'v1SidebarWidthExpanded',
    defaultExpandedWidth,
  );
  const [storedCollapsed, setStoredCollapsed] = useLocalStorageState(
    'v1SidebarCollapsed',
    false,
  );
  // Initialize from the current viewport so we don't "animate" from mobile->desktop
  // width on first load (the resize effect will keep it updated).
  const [isWide, setIsWide] = useState(() =>
    typeof window !== 'undefined' ? window.innerWidth >= 768 : false,
  );
  const [isResizing, setIsResizing] = useState(false);
  const [liveWidth, setLiveWidth] = useState<number | null>(null);
  const [startX, setStartX] = useState(0);
  const [startWidth, setStartWidth] = useState(0);
  const wasCollapsedAtDragStartRef = useRef(false);
  const didDragRef = useRef(false);
  const sidebarRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleResize = () => {
      setIsWide(window.innerWidth >= 768);
    };

    handleResize();
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
    };
  }, []);

  const onNavLinkClick = useCallback(() => {
    if (window.innerWidth > 768) {
      return;
    }

    setSidebarOpen('closed');
  }, [setSidebarOpen]);

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

  const commonParams = useMemo(() => ({ tenant: tenantId }), [tenantId]);
  const isActive = useCallback(
    (to: string, fuzzy = false) =>
      Boolean(matchRoute({ to, params: commonParams, fuzzy })),
    [matchRoute, commonParams],
  );

  const managedWorkerEnabled = featureFlags?.data?.['managed-worker'];

  const navSections = useMemo<SidebarNavSection[]>(() => {
    const billingLabel = cloudMeta?.data.canBill
      ? 'Billing & Limits'
      : 'Resource Limits';

    const settingsChildren: SidebarNavChild[] = [
      {
        key: 'tenant-settings-overview',
        name: 'Overview',
        to: appRoutes.tenantSettingsOverviewRoute.to,
      },
      {
        key: 'tenant-settings-api-tokens',
        name: 'API Tokens',
        to: appRoutes.tenantSettingsApiTokensRoute.to,
      },
      {
        key: 'tenant-settings-github',
        name: 'Github',
        to: appRoutes.tenantSettingsGithubRoute.to,
      },
      {
        key: 'tenant-settings-members',
        name: 'Members',
        to: appRoutes.tenantSettingsMembersRoute.to,
      },
      {
        key: 'tenant-settings-billing-and-limits',
        name: billingLabel,
        to: appRoutes.tenantSettingsBillingRoute.to,
      },
      {
        key: 'tenant-settings-alerting',
        name: 'Alerting',
        to: appRoutes.tenantSettingsAlertingRoute.to,
      },
      {
        key: 'tenant-settings-ingestors',
        name: 'Ingestors',
        to: appRoutes.tenantSettingsIngestorsRoute.to,
      },
      {
        key: 'quickstart',
        name: 'Quickstart',
        to: appRoutes.tenantOnboardingGetStartedRoute.to,
      },
    ];

    const sections: SidebarNavSection[] = [
      {
        key: 'activity',
        title: 'Activity',
        itemsClassName: 'flex flex-col gap-y-1',
        items: [
          {
            key: 'runs',
            name: 'Runs',
            to: appRoutes.tenantRunsRoute.to,
            icon: ({ collapsed }: { collapsed: boolean }) => (
              <PlayIcon
                className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
              />
            ),
          },
          {
            key: 'events',
            name: 'Events',
            to: appRoutes.tenantEventsRoute.to,
            icon: ({ collapsed }: { collapsed: boolean }) => (
              <SquareActivityIcon
                className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
              />
            ),
          },
        ],
      },
      {
        key: 'triggers',
        title: 'Triggers',
        itemsClassName: 'space-y-1',
        items: [
          {
            key: 'scheduled',
            name: 'Scheduled Runs',
            to: appRoutes.tenantScheduledRoute.to,
            icon: ({ collapsed }: { collapsed: boolean }) => (
              <CalendarDaysIcon
                className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
              />
            ),
          },
          {
            key: 'crons',
            name: 'Cron Jobs',
            to: appRoutes.tenantCronJobsRoute.to,
            icon: ({ collapsed }: { collapsed: boolean }) => (
              <ClockIcon
                className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
              />
            ),
          },
          {
            key: 'webhooks',
            name: 'Webhooks',
            to: appRoutes.tenantWebhooksRoute.to,
            icon: ({ collapsed }: { collapsed: boolean }) => (
              <WebhookIcon
                className={collapsed ? 'size-5' : 'mr-2 h-4 w-4 shrink-0'}
              />
            ),
          },
        ],
      },
      {
        key: 'resources',
        title: 'Resources',
        itemsClassName: 'space-y-1',
        items: [
          {
            key: 'workers',
            name: 'Workers',
            to: appRoutes.tenantWorkersRoute.to,
            icon: ({ collapsed }: { collapsed: boolean }) => (
              <ServerStackIcon
                className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
              />
            ),
          },
          {
            key: 'workflows',
            name: 'Workflows',
            to: appRoutes.tenantWorkflowsRoute.to,
            icon: ({ collapsed }: { collapsed: boolean }) => (
              <Squares2X2Icon
                className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
              />
            ),
          },
          ...(managedWorkerEnabled
            ? [
                {
                  key: 'managed-compute',
                  name: 'Managed Compute',
                  to: appRoutes.tenantManagedWorkersRoute.to,
                  icon: ({ collapsed }: { collapsed: boolean }) => (
                    <CpuChipIcon
                      className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
                    />
                  ),
                },
              ]
            : []),
          {
            key: 'rate-limits',
            name: 'Rate Limits',
            to: appRoutes.tenantRateLimitsRoute.to,
            icon: ({ collapsed }: { collapsed: boolean }) => (
              <ScaleIcon
                className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
              />
            ),
          },
          {
            key: 'filters',
            name: 'Filters',
            to: appRoutes.tenantFiltersRoute.to,
            icon: ({ collapsed }: { collapsed: boolean }) => (
              <Filter
                className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
              />
            ),
          },
        ],
      },
      {
        key: 'settings',
        title: 'Settings',
        itemsClassName: 'space-y-1',
        items: [
          {
            key: 'tenant-settings',
            name: 'General',
            to: appRoutes.tenantSettingsOverviewRoute.to,
            activeTo: appRoutes.tenantSettingsIndexRoute.to,
            activeFuzzy: true,
            prefix: appRoutes.tenantSettingsIndexRoute.to,
            icon: ({ collapsed }: { collapsed: boolean }) => (
              <GearIcon
                className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
              />
            ),
            children: settingsChildren,
          },
        ],
      },
    ];

    return sections;
  }, [cloudMeta?.data.canBill, managedWorkerEnabled]);

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
        'relative absolute inset-x-0 top-0 bottom-0 z-[100] w-full overflow-hidden border-r bg-slate-100 dark:bg-slate-900 md:relative md:inset-auto md:top-0 md:bottom-auto md:h-full md:bg-[unset] md:dark:bg-[unset]',
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
          'absolute right-0 top-0 bottom-0 z-10 hidden w-1 cursor-col-resize transition-colors hover:bg-blue-500/20 md:block',
          isResizing && 'bg-blue-500/30',
        )}
        onMouseDown={handleMouseDown}
      />

      <div className="flex h-full flex-col overflow-hidden">
        {renderCollapsed ? (
          <div className="flex h-full flex-col items-center justify-between py-4">
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
                              className={cn(
                                'w-10',
                                active && 'bg-slate-200 dark:bg-slate-800',
                              )}
                            >
                              {item.icon({ collapsed: true })}
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent
                            side="right"
                            align="start"
                            className="bg-secondary text-secondary-foreground"
                          >
                            {item.children.map((child) => (
                              <DropdownMenuItem
                                key={child.key}
                                asChild
                                className="w-full cursor-pointer hover:bg-primary/10 focus:bg-primary/10"
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
                        {item.icon({ collapsed: true })}
                      </Button>
                    );
                  })}
                </React.Fragment>
              ))}
            </div>

            <Popover>
              <PopoverTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  hoverText={isCloudEnabled ? 'Organization' : 'Tenant'}
                  hoverTextSide="right"
                  className="w-10"
                >
                  <BuildingOffice2Icon className="size-5" />
                </Button>
              </PopoverTrigger>
              <PopoverContent side="right" align="start" className="w-80 p-2">
                {isCloudEnabled ? (
                  <OrganizationSelector memberships={memberships} />
                ) : (
                  <TenantSwitcher memberships={memberships} />
                )}
              </PopoverContent>
            </Popover>
          </div>
        ) : (
          <>
            {/* Scrollable navigation area (keep scrollbar flush to sidebar edge) */}
            <div
              data-cy="v1-sidebar-scroll"
              className="min-h-0 flex-1 overflow-auto [scrollbar-gutter:stable] scrollbar-thin scrollbar-track-transparent scrollbar-thumb-muted-foreground"
            >
              <div className="px-4 py-4">
                {navSections.map((section) => (
                  <div key={section.key} className="py-2">
                    <h2 className="mb-2 text-lg font-semibold tracking-tight">
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
                          icon={item.icon({ collapsed: false })}
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
              {isCloudEnabled ? (
                <OrganizationSelector memberships={memberships} />
              ) : (
                <TenantSwitcher memberships={memberships} />
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}

function SidebarButtonPrimary({
  onNavLinkClick,
  to,
  params,
  name,
  icon,
  prefix,
  collapsibleChildren = [],
}: {
  onNavLinkClick: () => void;
  to: string;
  params?: Record<string, string>;
  name: string;
  icon: React.ReactNode;
  prefix?: string;
  collapsibleChildren?: React.ReactNode[];
}) {
  const matchRoute = useMatchRoute();

  // `to` (and `prefix`) are TanStack route templates (e.g. `/tenants/$tenant/...`).
  // Use the router matcher instead of raw string comparisons against `location.pathname`.
  const open =
    collapsibleChildren.length > 0
      ? prefix
        ? Boolean(matchRoute({ to: prefix, params, fuzzy: true }))
        : Boolean(matchRoute({ to, params, fuzzy: true }))
      : false;

  const selected =
    collapsibleChildren.length > 0 ? open : Boolean(matchRoute({ to, params }));

  const primaryLink = (
    <Link to={to} params={params} onClick={onNavLinkClick}>
      <Button
        variant="ghost"
        className={cn(
          'w-full justify-start pl-2 min-w-0 overflow-hidden',
          selected && 'bg-slate-200 dark:bg-slate-800',
        )}
      >
        {icon}
        <span className="truncate">{name}</span>
      </Button>
    </Link>
  );

  return collapsibleChildren.length == 0 ? (
    primaryLink
  ) : (
    <Collapsible
      open={open}
      // onOpenChange={setIsOpen}
      className="w-full"
    >
      {primaryLink}
      <CollapsibleContent className={'ml-4 space-y-2 border-l border-muted'}>
        {collapsibleChildren}
      </CollapsibleContent>
    </Collapsible>
  );
}

function SidebarButtonSecondary({
  onNavLinkClick,
  to,
  params,
  name,
  prefix,
}: {
  onNavLinkClick: () => void;
  to: string;
  params?: Record<string, string>;
  name: string;
  prefix?: string;
}) {
  const matchRoute = useMatchRoute();
  const hasPrefix = prefix
    ? Boolean(matchRoute({ to: prefix, params, fuzzy: true }))
    : false;
  const selected = Boolean(matchRoute({ to, params })) || hasPrefix;

  return (
    <Link to={to} params={params} onClick={onNavLinkClick}>
      <Button
        variant="ghost"
        size="sm"
        className={cn(
          'my-[1px] ml-1 mr-3 w-[calc(100%-3px)] justify-start pl-3 pr-0 min-w-0 overflow-hidden',
          selected && 'bg-slate-200 dark:bg-slate-800',
        )}
      >
        <span className="truncate">{name}</span>
      </Button>
    </Link>
  );
}
