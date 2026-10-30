import { TenantRegionBadge } from '@/components/v1/molecules/nav-bar/tenant-region-badge';
import {
  getHighlightRanges,
  matchesSearch,
  parseSearchQuery,
} from '@/components/v1/molecules/nav-bar/tenant-switcher-search';
import { Avatar, AvatarFallback } from '@/components/v1/ui/avatar';
import {
  Command,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/v1/ui/command';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { Spinner } from '@/components/v1/ui/loading';
import { Tenant } from '@/lib/api';
import { formatShardDeploymentKey } from '@/lib/shard-deployment-key';
import { cn } from '@/lib/utils';
import {
  ArrowTopRightOnSquareIcon,
  CheckIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  Cog6ToothIcon,
  DocumentDuplicateIcon,
  EllipsisHorizontalIcon,
  PlusIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline';
import { detectPlatform, formatForDisplay } from '@tanstack/react-hotkeys';
import React, { useMemo, useRef, useState } from 'react';

// Orgs with more tenants than this show only the first ones plus "Show more",
// and start collapsed unless they hold the current tenant.
const ORG_VISIBLE_TENANTS = 6;

const isMac = detectPlatform() === 'mac';

export type SwitcherOrganization = {
  id: string;
  name: string;
  isOwner: boolean;
  tenants: Tenant[];
};

export type SwitcherFooterAction = {
  id: string;
  label: string;
  icon: React.ReactNode;
  onSelect: () => void;
  dataCy?: string;
};

interface TenantSwitcherPanelProps {
  /** Tenants grouped by organization, in display order. Omit for a flat list. */
  organizations?: SwitcherOrganization[];
  tenants?: Tenant[];
  currentTenant?: Tenant;
  currentOrganizationId?: string;
  recentTenantIds: string[];
  isLoading?: boolean;
  footerActions: SwitcherFooterAction[];
  getTenantHref: (tenant: Tenant) => string;
  onSelectTenant: (tenant: Tenant) => void;
  onTenantSettings: (tenant: Tenant) => void;
  onOrganizationNewTenant?: (organizationId: string) => void;
  onOrganizationSettings?: (organizationId: string) => void;
}

// cmdk identifies items by lowercased value, so keys are built from IDs only.
type NavItem =
  | { kind: 'tenant'; key: string; tenant: Tenant; organizationId?: string }
  | { kind: 'organization'; key: string; organizationId: string }
  | { kind: 'more'; key: string; organizationId: string }
  | { kind: 'action'; key: string; action: SwitcherFooterAction };

type Section = {
  organization?: SwitcherOrganization;
  isOpen: boolean;
  rows: Tenant[];
  hiddenCount: number;
};

const tenantKey = (tenantId: string, organizationId?: string) =>
  `tenant:${organizationId ?? ''}:${tenantId}`;
const recentKey = (tenantId: string) => `recent:${tenantId}`;
const organizationKey = (organizationId: string) => `org:${organizationId}`;
const moreKey = (organizationId: string) => `more:${organizationId}`;
const actionKey = (actionId: string) => `action:${actionId}`;

const byName = (a: Tenant, b: Tenant) =>
  a.name.toLowerCase().localeCompare(b.name.toLowerCase());

function initials(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((word) => word[0].toUpperCase())
    .join('');
}

function Highlighted({ text, words }: { text: string; words: string[] }) {
  const ranges = getHighlightRanges(text, words);
  if (ranges.length === 0) {
    return <>{text}</>;
  }

  const parts: React.ReactNode[] = [];
  let cursor = 0;
  ranges.forEach((range) => {
    parts.push(text.slice(cursor, range.start));
    parts.push(
      <mark
        key={range.start}
        className="rounded-sm bg-yellow-200 text-inherit dark:bg-yellow-500/40"
      >
        {text.slice(range.start, range.end)}
      </mark>,
    );
    cursor = range.end;
  });
  parts.push(text.slice(cursor));
  return <>{parts}</>;
}

function Kbd({ children }: { children: React.ReactNode }) {
  return (
    <kbd className="rounded border border-b-2 bg-background px-1 font-mono text-[10px]">
      {children}
    </kbd>
  );
}

export function TenantSwitcherPanel({
  organizations,
  tenants,
  currentTenant,
  currentOrganizationId,
  recentTenantIds,
  isLoading,
  footerActions,
  getTenantHref,
  onSelectTenant,
  onTenantSettings,
  onOrganizationNewTenant,
  onOrganizationSettings,
}: TenantSwitcherPanelProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState('');
  const [activeKey, setActiveKey] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<Set<string>>(
    () =>
      new Set(
        (organizations ?? [])
          .filter(
            (org) =>
              org.id === currentOrganizationId ||
              org.tenants.length <= ORG_VISIBLE_TENANTS,
          )
          .map((org) => org.id),
      ),
  );
  const [showAll, setShowAll] = useState<Set<string>>(() => new Set());

  const words = useMemo(() => parseSearchQuery(query), [query]);
  const isSearching = words.length > 0;
  const currentTenantId = currentTenant?.metadata.id;

  const organizationByTenantId = useMemo(() => {
    const map = new Map<string, SwitcherOrganization>();
    organizations?.forEach((org) =>
      org.tenants.forEach((tenant) => map.set(tenant.metadata.id, org)),
    );
    return map;
  }, [organizations]);

  const recents = useMemo(() => {
    if (isSearching) {
      return [];
    }
    const all = organizations
      ? organizations.flatMap((org) => org.tenants)
      : (tenants ?? []);
    return recentTenantIds
      .map((id) => all.find((tenant) => tenant.metadata.id === id))
      .filter((tenant): tenant is Tenant => !!tenant);
  }, [isSearching, organizations, tenants, recentTenantIds]);

  const sections: Section[] = useMemo(() => {
    const tenantMatches = (tenant: Tenant, organizationName?: string) =>
      matchesSearch(words, [
        tenant.name,
        tenant.slug,
        tenant.region,
        formatShardDeploymentKey(tenant.region),
        organizationName,
      ]);

    if (!organizations) {
      const rows = (tenants ?? [])
        .filter((tenant) => tenantMatches(tenant))
        .sort(byName);
      return [{ isOpen: true, rows, hiddenCount: 0 }];
    }

    return organizations.flatMap((org): Section[] => {
      let rows = org.tenants
        .filter((tenant) => tenantMatches(tenant, org.name))
        .sort(byName);

      if (isSearching && rows.length === 0) {
        return [];
      }

      let hiddenCount = 0;
      if (
        !isSearching &&
        !showAll.has(org.id) &&
        rows.length > ORG_VISIBLE_TENANTS
      ) {
        let visible = rows.slice(0, ORG_VISIBLE_TENANTS);
        const current = rows.find(
          (tenant) => tenant.metadata.id === currentTenantId,
        );
        if (current && !visible.includes(current)) {
          visible = [...visible.slice(0, ORG_VISIBLE_TENANTS - 1), current];
        }
        hiddenCount = rows.length - visible.length;
        rows = visible;
      }

      return [
        {
          organization: org,
          isOpen: isSearching || expanded.has(org.id),
          rows,
          hiddenCount,
        },
      ];
    });
  }, [
    organizations,
    tenants,
    words,
    isSearching,
    expanded,
    showAll,
    currentTenantId,
  ]);

  const matchCount = sections.reduce(
    (sum, section) => sum + section.rows.length,
    0,
  );

  // Every selectable row in display order, for keyboard handling and the
  // default selection.
  const itemsByKey = useMemo(() => {
    const result: NavItem[] = recents.map((tenant) => ({
      kind: 'tenant',
      key: recentKey(tenant.metadata.id),
      tenant,
    }));
    sections.forEach((section) => {
      const orgId = section.organization?.id;
      if (orgId) {
        result.push({
          kind: 'organization',
          key: organizationKey(orgId),
          organizationId: orgId,
        });
      }
      if (!section.isOpen) {
        return;
      }
      section.rows.forEach((tenant) =>
        result.push({
          kind: 'tenant',
          key: tenantKey(tenant.metadata.id, orgId),
          tenant,
          organizationId: orgId,
        }),
      );
      if (orgId && section.hiddenCount > 0) {
        result.push({
          kind: 'more',
          key: moreKey(orgId),
          organizationId: orgId,
        });
      }
    });
    footerActions.forEach((action) =>
      result.push({ kind: 'action', key: actionKey(action.id), action }),
    );
    return new Map(result.map((item) => [item.key, item]));
  }, [recents, sections, footerActions]);

  // With no explicit choice, preselect the most recent tenant (so open + Enter
  // goes back to it), or the first match while searching.
  const defaultKey = useMemo(() => {
    const keys = [...itemsByKey.keys()];
    if (isSearching) {
      return keys.find((key) => itemsByKey.get(key)?.kind === 'tenant');
    }
    if (recents[0]) {
      return recentKey(recents[0].metadata.id);
    }
    if (currentTenantId) {
      const key = tenantKey(
        currentTenantId,
        organizationByTenantId.get(currentTenantId)?.id,
      );
      if (itemsByKey.has(key)) {
        return key;
      }
    }
    return keys[0];
  }, [
    isSearching,
    itemsByKey,
    recents,
    currentTenantId,
    organizationByTenantId,
  ]);

  const selectedKey =
    activeKey && itemsByKey.has(activeKey) ? activeKey : (defaultKey ?? '');

  const setOrganizationOpen = (organizationId: string, open: boolean) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (open) {
        next.add(organizationId);
      } else {
        next.delete(organizationId);
      }
      return next;
    });
  };

  const openInNewTab = (tenant: Tenant) => {
    window.open(getTenantHref(tenant), '_blank', 'noopener');
  };

  // ↑↓ and Enter are handled by cmdk; this adds the tree keys and ⌘/Ctrl+Enter.
  const onKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    const active = itemsByKey.get(selectedKey);
    if (!active) {
      return;
    }

    if (e.key === 'Enter' && (isMac ? e.metaKey : e.ctrlKey)) {
      if (active.kind === 'tenant') {
        e.preventDefault();
        openInNewTab(active.tenant);
      }
      return;
    }

    if (e.key === 'ArrowRight' && active.kind === 'organization') {
      e.preventDefault();
      setOrganizationOpen(active.organizationId, true);
      return;
    }

    if (e.key === 'ArrowLeft') {
      if (active.kind === 'organization') {
        e.preventDefault();
        setOrganizationOpen(active.organizationId, false);
      } else if (
        // While typing, ← keeps moving the caret.
        !query &&
        (active.kind === 'tenant' || active.kind === 'more') &&
        active.organizationId
      ) {
        e.preventDefault();
        setActiveKey(organizationKey(active.organizationId));
      }
    }
  };

  const renderTenantRow = (tenant: Tenant, key: string, isRecent = false) => {
    const isCurrent = tenant.metadata.id === currentTenantId;
    const organization = organizationByTenantId.get(tenant.metadata.id);

    return (
      <CommandItem
        key={key}
        value={key}
        onSelect={() => onSelectTenant(tenant)}
        aria-current={isCurrent ? 'true' : undefined}
        data-cy={`tenant-switcher-${isRecent ? 'recent' : 'item'}-${tenant.slug}`}
        className={cn(
          'group cursor-pointer gap-2',
          organization && !isRecent && 'pl-9',
        )}
      >
        <span className="flex size-5 shrink-0 items-center justify-center">
          <span className="size-2 rounded-full bg-green-500" />
        </span>
        <span
          className={cn(
            'line-clamp-2 min-w-0 flex-1 break-words leading-snug',
            isCurrent && 'font-semibold',
          )}
        >
          <Highlighted text={tenant.name} words={words} />
        </span>
        {isRecent ? (
          organization && (
            <span className="shrink-0 whitespace-nowrap text-xs text-muted-foreground">
              {organization.name}
            </span>
          )
        ) : (
          <TenantRegionBadge region={tenant.region} />
        )}
        <span className="flex w-4 shrink-0 justify-center">
          {isCurrent && <CheckIcon className="size-4" />}
        </span>
        {!isRecent && (
          <TenantRowMenu
            tenant={tenant}
            onTenantSettings={onTenantSettings}
            onOpenInNewTab={openInNewTab}
            onClose={() => inputRef.current?.focus()}
          />
        )}
      </CommandItem>
    );
  };

  const renderOrganization = (
    section: Section & { organization: SwitcherOrganization },
  ) => {
    const org = section.organization;
    return (
      <React.Fragment key={org.id}>
        {/* While searching every org is open and its header is a label, so
            cmdk's own "select the first item" lands on the first match. */}
        <CommandItem
          value={organizationKey(org.id)}
          disabled={isSearching}
          onSelect={() => setOrganizationOpen(org.id, !section.isOpen)}
          aria-expanded={isSearching ? undefined : section.isOpen}
          className="cursor-pointer gap-2 data-[disabled]:pointer-events-auto data-[disabled]:cursor-default data-[disabled]:opacity-100"
        >
          <span className="flex size-5 shrink-0 items-center justify-center text-muted-foreground">
            {!isSearching &&
              (section.isOpen ? (
                <ChevronDownIcon className="size-3" />
              ) : (
                <ChevronRightIcon className="size-3" />
              ))}
          </span>
          <Avatar aria-hidden className="size-5 rounded">
            <AvatarFallback className="rounded text-[10px] font-semibold">
              {initials(org.name)}
            </AvatarFallback>
          </Avatar>
          <span className="min-w-0 truncate font-semibold" title={org.name}>
            <Highlighted text={org.name} words={words} />
          </span>
          {org.isOwner && (
            <span className="ml-auto flex shrink-0 items-center gap-0.5">
              <OrganizationButton
                label={`New tenant in ${org.name}`}
                title="New tenant"
                onClick={() => onOrganizationNewTenant?.(org.id)}
              >
                <PlusIcon className="size-3.5" />
              </OrganizationButton>
              <OrganizationButton
                label={`${org.name} settings`}
                title="Organization settings"
                onClick={() => onOrganizationSettings?.(org.id)}
              >
                <Cog6ToothIcon className="size-3.5" />
              </OrganizationButton>
            </span>
          )}
        </CommandItem>
        {section.isOpen &&
          section.rows.map((tenant) =>
            renderTenantRow(tenant, tenantKey(tenant.metadata.id, org.id)),
          )}
        {section.isOpen && section.hiddenCount > 0 && (
          <CommandItem
            value={moreKey(org.id)}
            onSelect={() => setShowAll((prev) => new Set(prev).add(org.id))}
            className="cursor-pointer pl-16 text-xs text-muted-foreground"
          >
            Show {section.hiddenCount} more
          </CommandItem>
        )}
      </React.Fragment>
    );
  };

  const organizationSections = sections.filter(
    (section): section is Section & { organization: SwitcherOrganization } =>
      !!section.organization,
  );
  const flatSection = organizations ? undefined : sections[0];

  return (
    <Command
      value={selectedKey}
      onValueChange={setActiveKey}
      shouldFilter={false}
      // Ctrl+K opens the switcher on Windows and Linux.
      vimBindings={false}
      onKeyDown={onKeyDown}
      className="max-h-[min(40rem,calc(100vh-6rem))] rounded-none"
    >
      <div className="relative">
        <CommandInput
          ref={inputRef}
          autoFocus
          value={query}
          onValueChange={(value) => {
            setQuery(value);
            setActiveKey(null);
          }}
          placeholder={
            organizations ? 'Find a tenant or organization…' : 'Find a tenant…'
          }
          aria-label={
            organizations
              ? 'Search tenants and organizations'
              : 'Search tenants'
          }
          className="pr-8"
        />
        {query && (
          <button
            type="button"
            aria-label="Clear search"
            onClick={() => {
              setQuery('');
              setActiveKey(null);
              inputRef.current?.focus();
            }}
            className="absolute right-2 top-1/2 flex size-6 -translate-y-1/2 items-center justify-center rounded text-muted-foreground hover:bg-accent hover:text-foreground"
          >
            <XMarkIcon className="size-3.5" />
          </button>
        )}
      </div>

      <CommandList
        data-cy="tenant-switcher-list"
        className="max-h-none min-h-0 flex-1"
      >
        {isLoading && (
          <div className="flex items-center gap-2 px-3 py-2 text-sm text-muted-foreground">
            <Spinner className="mr-0" />
            Loading organizations…
          </div>
        )}

        {recents.length > 0 && (
          <CommandGroup heading="Recent">
            {recents.map((tenant) =>
              renderTenantRow(tenant, recentKey(tenant.metadata.id), true),
            )}
          </CommandGroup>
        )}

        {flatSection && flatSection.rows.length > 0 && (
          <CommandGroup
            heading={
              isSearching
                ? `${matchCount} ${matchCount === 1 ? 'match' : 'matches'}`
                : recents.length > 0
                  ? 'All tenants'
                  : undefined
            }
          >
            {flatSection.rows.map((tenant) =>
              renderTenantRow(tenant, tenantKey(tenant.metadata.id)),
            )}
          </CommandGroup>
        )}

        {isSearching && organizationSections.length > 0 && (
          <CommandGroup
            heading={`${matchCount} ${matchCount === 1 ? 'match' : 'matches'}`}
          >
            {organizationSections.map(renderOrganization)}
          </CommandGroup>
        )}

        {!isSearching &&
          organizationSections
            .filter(
              (section) => section.organization.id === currentOrganizationId,
            )
            .map((section) => (
              <CommandGroup
                key={section.organization.id}
                heading="Current organization"
              >
                {renderOrganization(section)}
              </CommandGroup>
            ))}

        {!isSearching &&
          organizationSections.some(
            (section) => section.organization.id !== currentOrganizationId,
          ) && (
            <CommandGroup heading="Other organizations">
              {organizationSections
                .filter(
                  (section) =>
                    section.organization.id !== currentOrganizationId,
                )
                .map(renderOrganization)}
            </CommandGroup>
          )}

        {isSearching && matchCount === 0 && !isLoading && (
          <div className="flex flex-col items-center gap-2 px-5 py-6 text-center">
            <span className="text-sm font-medium">
              No tenants match “{query.trim()}”
            </span>
            <span className="text-xs text-muted-foreground">
              {organizations
                ? 'Search covers tenant names, slugs, regions and organization names.'
                : 'Search covers tenant names, slugs and regions.'}
            </span>
          </div>
        )}
      </CommandList>

      {footerActions.length > 0 && (
        <CommandGroup className="shrink-0 border-t">
          {footerActions.map((action) => (
            <CommandItem
              key={action.id}
              value={actionKey(action.id)}
              onSelect={action.onSelect}
              data-cy={action.dataCy}
              className="cursor-pointer gap-2"
            >
              {action.icon}
              {action.label}
            </CommandItem>
          ))}
        </CommandGroup>
      )}

      <div className="flex shrink-0 items-center gap-3 border-t bg-muted/40 px-3 py-1.5 text-xs text-muted-foreground">
        <span>
          <Kbd>↑</Kbd> <Kbd>↓</Kbd> move
        </span>
        <span>
          <Kbd>↵</Kbd> switch
        </span>
        <span>
          <Kbd>{formatForDisplay('Mod')}</Kbd> <Kbd>↵</Kbd> new tab
        </span>
        <span className="ml-auto">
          <Kbd>esc</Kbd>
        </span>
      </div>
    </Command>
  );
}

function OrganizationButton({
  label,
  title,
  onClick,
  children,
}: {
  label: string;
  title: string;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      title={title}
      aria-label={label}
      onClick={(e) => {
        // Keep the click from also toggling the org row.
        e.stopPropagation();
        onClick();
      }}
      className="flex size-6 items-center justify-center rounded text-muted-foreground hover:bg-foreground/10 hover:text-foreground"
    >
      {children}
    </button>
  );
}

function TenantRowMenu({
  tenant,
  onTenantSettings,
  onOpenInNewTab,
  onClose,
}: {
  tenant: Tenant;
  onTenantSettings: (tenant: Tenant) => void;
  onOpenInNewTab: (tenant: Tenant) => void;
  onClose: () => void;
}) {
  const region = formatShardDeploymentKey(tenant.region);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={`More actions for ${tenant.name}`}
          title="More"
          onClick={(e) => e.stopPropagation()}
          className="absolute right-1 top-1/2 flex size-6 -translate-y-1/2 items-center justify-center rounded bg-accent text-muted-foreground opacity-0 shadow-[-10px_0_8px_hsl(var(--accent))] hover:bg-foreground/10 hover:text-foreground focus-visible:opacity-100 group-hover:opacity-100 data-[state=open]:opacity-100"
        >
          <EllipsisHorizontalIcon className="size-4" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="w-56"
        // The menu is portalled but React events still bubble to the cmdk
        // root, where Enter or a click would also select the tenant row.
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => e.stopPropagation()}
        // Return focus to the search input so keyboard navigation continues.
        onCloseAutoFocus={(e) => {
          e.preventDefault();
          onClose();
        }}
      >
        <div className="px-2 pb-2 pt-1.5">
          <div className="break-words text-xs font-medium">{tenant.name}</div>
          <div className="mt-0.5 font-mono text-xs text-muted-foreground">
            {region ? `${region} · ` : ''}slug {tenant.slug}
          </div>
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={() => onTenantSettings(tenant)}>
          <Cog6ToothIcon className="mr-2 size-4" />
          Tenant settings
        </DropdownMenuItem>
        <DropdownMenuItem
          onSelect={() => {
            void navigator.clipboard.writeText(tenant.metadata.id);
          }}
        >
          <DocumentDuplicateIcon className="mr-2 size-4" />
          Copy tenant ID
        </DropdownMenuItem>
        <DropdownMenuItem onSelect={() => onOpenInNewTab(tenant)}>
          <ArrowTopRightOnSquareIcon className="mr-2 size-4" />
          Open in new tab
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
