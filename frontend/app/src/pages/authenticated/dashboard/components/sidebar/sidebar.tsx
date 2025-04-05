'use client';

import {
  BadgeCheck,
  ChevronRight,
  ChevronsUpDown,
  LogOut,
  MessageCircle,
  Moon,
  Plus,
  Sun,
} from 'lucide-react';

import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
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
} from '@/components/ui/sidebar';
import { PropsWithChildren } from 'react';
import useUser from '@/hooks/use-user';
import { getMainNavLinks } from './main-nav';
import { TenantBlock, UserBlock } from './user-dropdown';
import { useTheme } from '@/components/theme-provider';
import useApiMeta from '@/hooks/use-api-meta';
import useSupportChat from '@/hooks/use-support-chat';
import { QuestionMarkCircledIcon } from '@radix-ui/react-icons';
import useTenant from '@/hooks/use-tenant';
import { useNavigate, useLocation, Link } from 'react-router-dom';
export const iframeHeight = '800px';

export const description = 'An inset sidebar with secondary navigation.';

export function AppSidebar({ children }: PropsWithChildren) {
  const meta = useApiMeta();
  const { data: user, memberships, logout } = useUser();
  const { toggleTheme, theme } = useTheme();
  const chat = useSupportChat();
  const { setTenant } = useTenant();
  const navigate = useNavigate();
  const location = useLocation();
  const navLinks = getMainNavLinks(location.pathname);

  return (
    <>
      <Sidebar variant="inset" collapsible="icon">
        <SidebarHeader>
          <SidebarMenu>
            <header className="flex h-16 shrink-0 items-center gap-2">
              <SidebarMenuItem>
                <SidebarMenuButton>
                  🪓 <span>Hatchet</span>
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
                  <DropdownMenuItem onClick={() => navigate('/onboarding/new')}>
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
                    defaultOpen={item.isActive}
                  >
                    <SidebarMenuItem>
                      <SidebarMenuButton asChild tooltip={item.title}>
                        <Link to={item.url}>
                          <item.icon />
                          <span>{item.title}</span>
                        </Link>
                      </SidebarMenuButton>
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
                                  <SidebarMenuSubButton asChild>
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
                        <pre className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                          <code>
                            ver: {meta?.version}
                            <br />
                            userId: {user?.metadata.id}
                            <br />
                            email: {user?.email}
                            <br />
                            name: {user?.name}
                          </code>
                        </pre>
                      </DropdownMenuLabel>
                    </DropdownMenuContent>
                  </DropdownMenu>
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
          <SidebarMenu>
            <SidebarMenuItem>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <SidebarMenuButton
                    size="lg"
                    className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                  >
                    <UserBlock />
                    <ChevronsUpDown className="ml-auto size-4" />
                  </SidebarMenuButton>
                </DropdownMenuTrigger>
                <DropdownMenuContent
                  className="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg"
                  side="right"
                  align="end"
                  sideOffset={4}
                >
                  <DropdownMenuLabel className="p-0 font-normal">
                    <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                      <UserBlock />
                    </div>
                  </DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuGroup>
                    <DropdownMenuItem>
                      {/* TODO: Add account settings page */}
                      <BadgeCheck />
                      Account Settings
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => toggleTheme()}>
                      {theme === 'dark' ? <Moon /> : <Sun />}
                      Toggle Theme
                    </DropdownMenuItem>
                  </DropdownMenuGroup>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={() => logout.mutate()}>
                    <LogOut />
                    Log out
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarFooter>
      </Sidebar>
      <SidebarInset>{children}</SidebarInset>
    </>
  );
}
