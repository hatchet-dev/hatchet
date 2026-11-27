import {
  SwitcherOrganization,
  TenantSwitcherPanel,
} from './tenant-switcher-panel';
import { Tenant, TenantVersion } from '@/lib/api';
import { PlusIcon } from '@heroicons/react/24/outline';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { detectPlatform } from '@tanstack/react-hotkeys';
import { expect, fn, spyOn, userEvent, waitFor, within } from 'storybook/test';

const tenant = (slug: string, region: string, name = slug): Tenant => ({
  metadata: {
    id: `tenant-${slug}`,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  },
  name,
  slug,
  region,
  version: TenantVersion.V1,
});

const currentOrg: SwitcherOrganization = {
  id: 'org-current',
  name: 'Current org',
  isOwner: true,
  tenants: [
    tenant('control-plane', 'aws:us-east-1'),
    tenant('development', 'aws:us-west-2'),
    tenant('staging', 'aws:us-east-1'),
    tenant('production', 'aws:us-east-1'),
    tenant('prod-eu', 'aws:eu-west-1', 'production-eu'),
    tenant(
      'long-name',
      'aws:eu-west-1',
      'shared-sandbox-tenant-with-a-long-descriptive-name',
    ),
    tenant('perf-bench', 'aws:us-east-1:dedicated-shard-with-a-long-name'),
    tenant('docs-examples', 'aws:us-east-1'),
    tenant('ltr', 'aws:us-west-2', 'load-test-replay'),
    tenant('sdk-ci-py', 'aws:us-east-1', 'sdk-ci-python'),
    tenant('sdk-ci-ts', 'aws:us-east-1', 'sdk-ci-typescript'),
  ],
};

const smallOrg: SwitcherOrganization = {
  id: 'org-small',
  name: 'Small org',
  isOwner: false,
  tenants: [
    tenant('small-sandbox', 'aws:us-west-2:dedicated-shard', 'sandbox'),
    tenant('small-staging', 'aws:us-west-2', 'staging-secondary'),
  ],
};

const largeOrg: SwitcherOrganization = {
  id: 'org-large',
  name: 'Large org with a long organization name',
  isOwner: true,
  tenants: [
    tenant('large-prod-us', 'aws:us-east-1'),
    tenant('large-prod-eu', 'aws:eu-central-1'),
    tenant('large-staging', 'aws:us-east-1'),
    tenant('large-dev', 'aws:us-east-1'),
    tenant('large-sandbox', 'aws:us-east-1'),
    tenant('large-perf', 'aws:us-east-1'),
    tenant('large-qa', 'aws:us-east-1'),
  ],
};

const organizations = [currentOrg, smallOrg, largeOrg];
const currentTenant = currentOrg.tenants[0];

const meta = {
  title: 'Molecules/TenantSwitcherPanel',
  component: TenantSwitcherPanel,
  args: {
    organizations,
    currentTenant,
    currentOrganizationId: currentOrg.id,
    recentTenantIds: [
      'tenant-production',
      'tenant-small-sandbox',
      'tenant-staging',
    ],
    footerActions: [
      {
        id: 'create-organization',
        label: 'Create organization',
        icon: <PlusIcon className="size-3.5" />,
        onSelect: fn(),
      },
    ],
    getTenantHref: (t) => `/tenants/${t.metadata.id}/runs`,
    onSelectTenant: fn(),
    onTenantSettings: fn(),
    onOrganizationNewTenant: fn(),
    onOrganizationSettings: fn(),
  },
  render: (args) => (
    <div className="w-[420px] overflow-hidden rounded-xl border bg-popover text-popover-foreground shadow-lg">
      <TenantSwitcherPanel {...args} />
    </div>
  ),
} satisfies Meta<typeof TenantSwitcherPanel>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Searching: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.type(canvas.getByRole('combobox'), 'prod eu');
    await expect(canvas.getByText('2 matches')).toBeInTheDocument();
  },
};

export const SearchingAcrossOrganizations: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.type(canvas.getByRole('combobox'), 'STAGING');
    await expect(canvas.getByText('3 matches')).toBeInTheDocument();
  },
};

// Tenants on a named shard get a `provider:region:shard` region, shown as
// "region / shard". The shard name is searchable.
export const SearchingByShardName: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.type(canvas.getByRole('combobox'), 'dedicated shard');
    await expect(canvas.getByText('2 matches')).toBeInTheDocument();
  },
};

export const SearchingByRegionAndShard: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.type(canvas.getByRole('combobox'), 'west-2 dedicated');
    await expect(canvas.getByText('1 match')).toBeInTheDocument();
    await expect(canvas.getByText('sandbox')).toBeInTheDocument();
  },
};

export const NoResults: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.type(canvas.getByRole('combobox'), 'stg-eu');
    await expect(
      canvas.getByText('No tenants match “stg-eu”'),
    ).toBeInTheDocument();
  },
};

export const NoRecents: Story = {
  args: {
    recentTenantIds: [],
  },
};

export const LoadingOrganizations: Story = {
  args: {
    organizations: [],
    isLoading: true,
    recentTenantIds: [],
  },
};

const manyOrganizations: SwitcherOrganization[] = [
  currentOrg,
  smallOrg,
  largeOrg,
  ...[14, 3, 9, 22, 5, 11].map((count, index): SwitcherOrganization => {
    const n = index + 1;
    return {
      id: `org-${n}`,
      name: `Organization ${n}`,
      isOwner: n !== 2,
      tenants: Array.from({ length: count }, (_, i) =>
        tenant(
          `org-${n}-${i + 1}`,
          i % 3 === 0 ? 'aws:eu-west-1' : 'aws:us-east-1',
          `org-${n}-tenant-${i + 1}`,
        ),
      ),
    };
  }),
];

export const ManyOrganizations: Story = {
  args: {
    organizations: manyOrganizations,
    recentTenantIds: [
      'tenant-org-4-4',
      'tenant-small-sandbox',
      'tenant-org-1-2',
    ],
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const headers = canvasElement.querySelectorAll(
      '[role="option"][aria-expanded]',
    );
    await expect(headers).toHaveLength(manyOrganizations.length);
    const expanded = [...headers].filter(
      (header) => header.getAttribute('aria-expanded') === 'true',
    );
    await expect(expanded).toHaveLength(1);
    await expect(expanded[0]).toHaveTextContent(currentOrg.name);
    await expect(canvas.getByText('org-4-tenant-4')).toBeInTheDocument();
  },
};

export const FlatList: Story = {
  name: 'Flat list (OSS)',
  args: {
    organizations: undefined,
    currentOrganizationId: undefined,
    tenants: currentOrg.tenants,
    recentTenantIds: ['tenant-production'],
    footerActions: [
      {
        id: 'new-tenant',
        label: 'New tenant',
        icon: <PlusIcon className="size-3.5" />,
        onSelect: fn(),
      },
    ],
  },
};

const isMac = detectPlatform() === 'mac';

export const EnterReturnsToRecentTenant: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByRole('combobox')).toHaveFocus();
    await userEvent.keyboard('{Enter}');
    await expect(args.onSelectTenant).toHaveBeenCalledTimes(1);
    await expect(args.onSelectTenant).toHaveBeenCalledWith(
      expect.objectContaining({ slug: 'production' }),
    );
  },
};

export const EnterPicksFirstMatch: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.type(canvas.getByRole('combobox'), 'prod eu');
    await expect(canvas.getByText('2 matches')).toBeInTheDocument();
    await userEvent.keyboard('{Enter}');
    await expect(args.onSelectTenant).toHaveBeenCalledTimes(1);
    await expect(args.onSelectTenant).toHaveBeenCalledWith(
      expect.objectContaining({ slug: 'prod-eu' }),
    );
  },
};

export const ArrowKeysWalkOrganizationTree: Story = {
  play: async ({ canvasElement, step }) => {
    const canvas = within(canvasElement);
    const currentOrg = canvas.getByRole('option', { name: /^Current org/ });

    await step('↓ past the three recent tenants', async () => {
      await userEvent.keyboard('{ArrowDown}{ArrowDown}{ArrowDown}');
      await expect(currentOrg).toHaveAttribute('aria-selected', 'true');
    });

    await step('← collapses and → expands the organization', async () => {
      await userEvent.keyboard('{ArrowLeft}');
      await expect(currentOrg).toHaveAttribute('aria-expanded', 'false');
      await userEvent.keyboard('{ArrowRight}');
      await expect(currentOrg).toHaveAttribute('aria-expanded', 'true');
    });

    await step('← on a tenant goes back to its organization', async () => {
      await userEvent.keyboard('{ArrowDown}');
      await expect(
        canvas.getByTestId('tenant-switcher-item-control-plane'),
      ).toHaveAttribute('aria-selected', 'true');
      await userEvent.keyboard('{ArrowLeft}');
      await expect(currentOrg).toHaveAttribute('aria-selected', 'true');
    });
  },
};

export const ModEnterOpensNewTab: Story = {
  name: 'Cmd/Ctrl+Enter opens a new tab',
  beforeEach: () => {
    const open = spyOn(window, 'open').mockReturnValue(null);
    return () => open.mockRestore();
  },
  play: async ({ args }) => {
    const modifier = isMac ? 'Meta' : 'Control';
    await userEvent.keyboard(`{${modifier}>}{Enter}{/${modifier}}`);
    await expect(window.open).toHaveBeenCalledTimes(1);
    await expect(window.open).toHaveBeenCalledWith(
      '/tenants/tenant-production/runs',
      '_blank',
      'noopener',
    );
    await expect(args.onSelectTenant).not.toHaveBeenCalled();
  },
};

export const ShowMore: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const rows = () => canvas.queryAllByTestId(/^tenant-switcher-item-/);
    const visibleBefore = rows().length;

    // Current org has 11 tenants and starts with 6 of them.
    await userEvent.click(canvas.getByRole('option', { name: 'Show 5 more' }));
    await expect(rows()).toHaveLength(visibleBefore + 5);
    await expect(
      canvas.getByTestId('tenant-switcher-item-sdk-ci-ts'),
    ).toBeInTheDocument();
  },
};

// The menu is portalled out of the panel, so it's queried from the document.
async function openRowMenu(canvasElement: HTMLElement, tenantName: string) {
  const canvas = within(canvasElement);
  await userEvent.click(
    canvas.getByRole('button', { name: `More actions for ${tenantName}` }),
  );
  return within(canvasElement.ownerDocument.body);
}

export const RowMenuOpensSettings: Story = {
  play: async ({ args, canvasElement }) => {
    const menu = await openRowMenu(canvasElement, 'development');
    await userEvent.click(
      await menu.findByRole('menuitem', { name: 'Tenant settings' }),
    );

    await expect(args.onTenantSettings).toHaveBeenCalledTimes(1);
    await expect(args.onTenantSettings).toHaveBeenCalledWith(
      expect.objectContaining({ slug: 'development' }),
    );
    await expect(args.onSelectTenant).not.toHaveBeenCalled();
    // Focus goes back to the search so keyboard navigation can continue.
    await waitFor(() =>
      expect(within(canvasElement).getByRole('combobox')).toHaveFocus(),
    );
  },
};

export const RowMenuCopiesTenantId: Story = {
  beforeEach: () => {
    const writeText = spyOn(
      navigator.clipboard,
      'writeText',
    ).mockResolvedValue();
    return () => writeText.mockRestore();
  },
  play: async ({ canvasElement }) => {
    const menu = await openRowMenu(canvasElement, 'development');
    await userEvent.click(
      await menu.findByRole('menuitem', { name: 'Copy tenant ID' }),
    );
    await expect(navigator.clipboard.writeText).toHaveBeenCalledTimes(1);
    await expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
      'tenant-development',
    );
  },
};
