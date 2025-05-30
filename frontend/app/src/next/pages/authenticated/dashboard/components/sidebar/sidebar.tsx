'use client';

import {
  CheckIcon,
  ChevronRight,
  ChevronsUpDown,
  MessageCircle,
  Plus,
} from 'lucide-react';

import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/next/components/ui/collapsible';
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/next/components/ui/command';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarRail,
  useSidebar,
} from '@/next/components/ui/sidebar';
import { PropsWithChildren, useMemo, useEffect, useState } from 'react';
import useUser from '@/next/hooks/use-user';
import { getMainNavLinks } from './main-nav';
import { TenantBlock } from './user-dropdown';
import useApiMeta from '@/next/hooks/use-api-meta';
import useSupportChat from '@/next/hooks/use-support-chat';
import { QuestionMarkCircledIcon } from '@radix-ui/react-icons';
import { useCurrentTenantId, useTenantDetails } from '@/next/hooks/use-tenant';
import { useNavigate, useLocation, Link } from 'react-router-dom';
import { Logo } from '@/next/components/ui/logo';
import { Code } from '@/next/components/ui/code';
import { ROUTES } from '@/next/lib/routes';
import {
  DropdownMenu,
  DropdownMenuItem,
  DropdownMenuGroup,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/next/components/ui/dropdown-menu';
import { DocsButton } from '@/next/components/ui/docs-button';
import docMetadata from '@/next/lib/docs';

export function AppSidebar({ children }: PropsWithChildren) {
  const meta = useApiMeta();
  const { data: user, memberships } = useUser();
  const chat = useSupportChat();
  const { setTenant } = useTenantDetails();
  const { tenantId } = useCurrentTenantId();
  const navigate = useNavigate();
  const location = useLocation();
  const navLinks = getMainNavLinks(tenantId, location.pathname);
  const { toggleSidebar, isCollapsed, isMobile, setOpenMobile } = useSidebar();

  const [collapsibleState, setCollapsibleState] = useState<
    Record<string, boolean>
  >({});
  const [openTenant, setOpenTenant] = useState(false);

  useEffect(() => {
    const savedState = localStorage.getItem('sidebar_collapsible_state');
    if (savedState) {
      try {
        setCollapsibleState(JSON.parse(savedState));
      } catch (error) {
        console.error('Failed to parse sidebar collapsible state', error);
      }
    }
  }, []);

  // Save collapsible state to localStorage
  const updateCollapsibleState = (key: string, isOpen: boolean) => {
    setCollapsibleState((prev) => {
      const newState = { ...prev, [key]: isOpen };
      localStorage.setItem(
        'sidebar_collapsible_state',
        JSON.stringify(newState),
      );
      return newState;
    });
  };

  const supportReference = useMemo(() => {
    return `ver: ${meta?.version}
tenantId: ${tenantId}
userId: ${user?.metadata.id}
email: ${user?.email}
name: ${user?.name}`;
  }, [meta, user, tenantId]);

  return (
    <>
      <Sidebar variant="sidebar" collapsible="icon">
        <SidebarRail />
        <SidebarHeader>
          <SidebarMenu>
            <header className="mb-3">
              <SidebarMenuItem>
                <SidebarMenuButton size="lg" onClick={() => toggleSidebar()}>
                  <Logo variant="md" />
                </SidebarMenuButton>
              </SidebarMenuItem>
            </header>
            <SidebarMenuItem>
              <SidebarMenuButton
                size="lg"
                className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                onClick={() => setOpenTenant(true)}
              >
                <TenantBlock />
                <ChevronsUpDown className="ml-auto size-4" />
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>
        <SidebarContent>
          {Object.entries(navLinks.sections).map(([key, section]) => (
            <SidebarGroup key={key}>
              <SidebarGroupLabel>{section.label}</SidebarGroupLabel>
              <SidebarMenu>
                {section.items.map((item) => (
                  <Collapsible
                    key={item.title}
                    asChild
                    defaultOpen={collapsibleState[item.title] ?? item.isActive}
                    onOpenChange={(isOpen) =>
                      updateCollapsibleState(item.title, isOpen)
                    }
                  >
                    <SidebarMenuItem>
                      {!isMobile && isCollapsed && item.items?.length ? (
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <SidebarMenuButton
                              tooltip={item.title}
                              className={
                                item.isActive
                                  ? 'bg-muted/50 hover:bg-muted/80'
                                  : 'hover:bg-muted/30'
                              }
                            >
                              <item.icon />
                              <span className="sr-only">{item.title}</span>
                            </SidebarMenuButton>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent
                            side="right"
                            sideOffset={4}
                            className="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg"
                          >
                            {item.items?.map((subItem) => (
                              <DropdownMenuItem key={subItem.title} asChild>
                                <Link
                                  to={subItem.url}
                                  onClick={() => setOpenMobile(false)}
                                >
                                  <subItem.icon className="mr-2 h-4 w-4" />
                                  {subItem.title}
                                </Link>
                              </DropdownMenuItem>
                            ))}
                          </DropdownMenuContent>
                        </DropdownMenu>
                      ) : (
                        <>
                          <CollapsibleTrigger asChild>
                            <SidebarMenuButton
                              asChild
                              tooltip={item.title}
                              className={
                                item.isActive
                                  ? 'bg-muted/50 hover:bg-muted/80'
                                  : 'hover:bg-muted/30'
                              }
                            >
                              <Link
                                to={item.url}
                                onClick={() => setOpenMobile(false)}
                              >
                                <item.icon />
                                <span>{item.title}</span>
                              </Link>
                            </SidebarMenuButton>
                          </CollapsibleTrigger>
                          {item.items?.length ? (
                            <>
                              <CollapsibleTrigger asChild>
                                <SidebarMenuAction className="data-[state=open]:rotate-90">
                                  <ChevronRight />
                                  <span className="sr-only">Toggle</span>
                                </SidebarMenuAction>
                              </CollapsibleTrigger>
                              <CollapsibleContent>
                                <SidebarMenuSub>
                                  {item.items?.map((subItem) => (
                                    <SidebarMenuSubItem key={subItem.title}>
                                      <SidebarMenuSubButton
                                        asChild
                                        className={
                                          subItem.isActive
                                            ? 'bg-muted/50 hover:bg-muted/80'
                                            : 'hover:bg-muted/30'
                                        }
                                      >
                                        <Link
                                          to={subItem.url}
                                          onClick={() => setOpenMobile(false)}
                                        >
                                          <subItem.icon />
                                          <span>{subItem.title}</span>
                                        </Link>
                                      </SidebarMenuSubButton>
                                    </SidebarMenuSubItem>
                                  ))}
                                </SidebarMenuSub>
                              </CollapsibleContent>
                            </>
                          ) : null}
                        </>
                      )}
                    </SidebarMenuItem>
                  </Collapsible>
                ))}
              </SidebarMenu>
            </SidebarGroup>
          ))}
          <SidebarGroup className="mt-auto">
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <SidebarMenuButton size="sm" tooltip="Support">
                        <QuestionMarkCircledIcon />
                        <span>Support</span>
                      </SidebarMenuButton>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent
                      className="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg"
                      side={isMobile ? 'bottom' : 'right'}
                      sideOffset={4}
                    >
                      <DropdownMenuGroup>
                        {chat.isEnabled() && (
                          <DropdownMenuItem onClick={() => chat.show()}>
                            <MessageCircle />
                            Chat with Support
                          </DropdownMenuItem>
                        )}
                        {navLinks.support.map((item) => (
                          <DropdownMenuItem
                            key={item.title}
                            onClick={() => {
                              window.open(item.url, item.target);
                            }}
                          >
                            <item.icon />
                            {item.title}
                          </DropdownMenuItem>
                        ))}
                      </DropdownMenuGroup>
                      <DropdownMenuSeparator />
                      <DropdownMenuLabel className="p-0 font-normal">
                        <Code
                          title="Support Reference"
                          language="text"
                          value={supportReference}
                        />
                      </DropdownMenuLabel>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </SidebarMenuItem>
                <SidebarMenuItem key="docs">
                  <DocsButton
                    doc={docMetadata.home.index}
                    prefix={''}
                    titleOverride="Documentation"
                    variant="ghost"
                    className="px-2 hover:bg-background"
                  />
                </SidebarMenuItem>
                {navLinks.navSecondary.map((item) => (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton asChild size="sm" tooltip={item.title}>
                      <Link
                        to={item.url}
                        target="_blank"
                        rel="noreferrer"
                        onClick={() => setOpenMobile(false)}
                      >
                        <item.icon />
                        <span>{item.title}</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>
        <SidebarFooter>
          {/* User dropdown has been moved to the main layout */}
        </SidebarFooter>
      </Sidebar>
      <SidebarInset>{children}</SidebarInset>
      <CommandDialog open={openTenant} onOpenChange={setOpenTenant}>
        <CommandInput placeholder="Switch tenants..." />
        <CommandList>
          <CommandEmpty>No tenants found.</CommandEmpty>
          <CommandGroup>
            {memberships
              ?.filter((membership) => !!membership.tenant)
              .sort((a, b) => a.tenant!.name.localeCompare(b.tenant!.name))
              .map((membership) => (
                <CommandItem
                  key={membership.tenant?.metadata.id}
                  onSelect={() => {
                    setTenant(membership.tenant!.metadata.id);
                    setOpenTenant(false);
                  }}
                >
                  {membership.tenant?.metadata.id === tenantId && (
                    <CheckIcon className="h-4 w-4 mr-2" />
                  )}
                  <span>{membership.tenant?.name}</span>
                </CommandItem>
              ))}
          </CommandGroup>
          <CommandGroup>
            <CommandItem
              onSelect={() => {
                navigate(ROUTES.onboarding.newTenant);
                setOpenTenant(false);
              }}
            >
              <Plus className="h-4 w-4 mr-2" />
              Create New Tenant
            </CommandItem>
          </CommandGroup>
        </CommandList>
      </CommandDialog>
    </>
  );
}
