import {
  SwitcherOrganization,
  TenantSwitcherPanel,
} from './tenant-switcher-panel';
import { Tenant, TenantVersion } from '@/lib/api';
import { PlusIcon } from '@heroicons/react/24/outline';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { expect, fn, userEvent, within } from 'storybook/test';

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
