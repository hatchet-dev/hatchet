'use client';

import {
  BookOpen,
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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/next/components/ui/dropdown-menu';
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
import useTenant from '@/next/hooks/use-tenant';
import { useNavigate, useLocation, Link } from 'react-router-dom';
import { Logo } from '@/next/components/ui/logo';
import { Code } from '@/next/components/ui/code';
import { pages, useDocs } from '@/next/hooks/use-docs-sheet';
import { ROUTES } from '@/next/lib/routes';
export const iframeHeight = '800px';

export const description = 'An inset sidebar with secondary navigation.';

export function AppSidebar({ children }: PropsWithChildren) {
  const meta = useApiMeta();
  const { data: user, memberships } = useUser();
  const chat = useSupportChat();
  const { tenant, setTenant } = useTenant();
  const navigate = useNavigate();
  const location = useLocation();
  const navLinks = getMainNavLinks(location.pathname);
  const { toggleSidebar } = useSidebar();
  const docs = useDocs();
  const [collapsibleState, setCollapsibleState] = useState<
    Record<string, boolean>
  >({});

  // Load collapsible state from localStorage on initial render
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
tenantId: ${tenant?.metadata.id}
userId: ${user?.metadata.id}
email: ${user?.email}
name: ${user?.name}`;
  }, [meta, tenant, user]);

  return (
    <>
      <Sidebar variant="floating" collapsible="icon">
        <SidebarRail />
        <SidebarHeader>
          <SidebarMenu>
            <header className="my-3">
              <SidebarMenuItem>
                <SidebarMenuButton size="lg" onClick={() => toggleSidebar()}>
                  <Logo variant="md" />
                </SidebarMenuButton>
              </SidebarMenuItem>
            </header>
            <SidebarMenuItem>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <SidebarMenuButton
                    size="lg"
                    className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                  >
                    <TenantBlock />
                    <ChevronsUpDown className="ml-auto size-4" />
                  </SidebarMenuButton>
                </DropdownMenuTrigger>
                <DropdownMenuContent
                  className="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg"
                  side="right"
                  sideOffset={4}
                >
                  <DropdownMenuGroup>
                    {memberships
                      ?.filter((membership) => !!membership.tenant)
                      .map((membership) => (
                        <DropdownMenuItem
                          key={membership.tenant?.metadata.id}
                          onClick={() => {
                            setTenant(membership.tenant!);
                          }}
                        >
                          <span>{membership.tenant?.name}</span>
                        </DropdownMenuItem>
                      ))}
                  </DropdownMenuGroup>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={() => navigate(ROUTES.onboarding.newTenant)}
                  >
                    <Plus />
                    Create New Tenant
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
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
                          <Link to={item.url}>
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
                                    <Link to={subItem.url}>
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
                      <SidebarMenuButton size="sm">
                        <QuestionMarkCircledIcon />
                        <span>Support</span>
                      </SidebarMenuButton>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent
                      className="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg"
                      side="right"
                      sideOffset={4}
                    >
                      <DropdownMenuGroup>
                        <DropdownMenuItem onClick={() => chat.show()}>
                          <MessageCircle />
                          Chat with Support
                        </DropdownMenuItem>
                        {navLinks.support.map((item) => (
                          <DropdownMenuItem key={item.title}>
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
                  <SidebarMenuButton
                    size="sm"
                    onClick={() => docs.toggle(pages.home.index)}
                  >
                    <BookOpen />
                    <span>Documentation</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
                {navLinks.navSecondary.map((item) => (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton asChild size="sm">
                      <Link to={item.url} target="_blank" rel="noreferrer">
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
    </>
  );
}
