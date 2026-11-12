import { LabsPreview, LabsPreviewProps } from './labs-preview';
import { Button } from '@/components/v1/ui/button';
import { cn } from '@/lib/utils';
import { settingsNavGroups } from '@/pages/main/v1/settings/settings-nav-items';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { ReactNode, useState } from 'react';
import { fn } from 'storybook/test';

function InteractiveLabs(props: LabsPreviewProps) {
  const [flags, setFlags] = useState(
    props.status === 'ready'
      ? props.flags
      : {
          durableStreams: false,
          sandboxOperator: false,
          serverlessOperator: false,
        },
  );

  if (props.status !== 'ready') {
    return <LabsPreview {...props} />;
  }

  return (
    <LabsPreview
      {...props}
      flags={flags}
      onToggle={(flag, enabled) => {
        props.onToggle(flag, enabled);
        setFlags((current) => ({ ...current, [flag]: enabled }));
      }}
    />
  );
}

function SettingsFrame({ children }: { children: ReactNode }) {
  const groups = settingsNavGroups({
    tenantId: 'tenant-1',
    orgId: 'org-1',
    canBill: true,
    isControlPlaneEnabled: true,
    isOrganizationOwner: true,
    canManageSso: true,
  });

  return (
    <div className="flex h-[720px] min-w-0 gap-x-6">
      <aside className="sticky top-0 w-56 shrink-0 self-start py-4">
        <h2 className="mb-4 px-2 text-lg font-semibold">Settings</h2>
        <div className="flex flex-col gap-y-6">
          {groups.map((group) => (
            <div key={group.key}>
              <h3 className="mb-2 px-2 text-xs font-mono tracking-widest uppercase text-muted-foreground">
                {group.title}
              </h3>
              <div className="flex flex-col gap-y-1">
                {group.items.map((item) => (
                  <Button
                    key={item.key}
                    variant="ghost"
                    className={cn(
                      'h-8 w-full justify-start pl-2 min-w-0 overflow-hidden',
                      item.key === 'labs' && 'bg-slate-200 dark:bg-slate-800',
                    )}
                  >
                    <span className="truncate">{item.name}</span>
                  </Button>
                ))}
              </div>
            </div>
          ))}
        </div>
      </aside>
      <div className="min-w-0 flex-1 overflow-auto">{children}</div>
    </div>
  );
}

const meta = {
  title: 'Pages/LabsPreview',
  component: LabsPreview,
} satisfies Meta<typeof LabsPreview>;

export default meta;

type Story = StoryObj<typeof meta>;

const off = {
  durableStreams: false,
  sandboxOperator: false,
  serverlessOperator: false,
};

export const AdminOff: Story = {
  args: {
    status: 'ready',
    flags: off,
    pending: false,
    onToggle: fn(),
  },
  render: (args) => <InteractiveLabs {...args} />,
};

export const AdminOn: Story = {
  args: {
    status: 'ready',
    flags: {
      durableStreams: true,
      sandboxOperator: false,
      serverlessOperator: true,
    },
    pending: false,
    onToggle: fn(),
  },
  render: (args) => <InteractiveLabs {...args} />,
};

export const Member: Story = {
  args: {
    status: 'gated',
  },
};

export const FreePlan: Story = {
  args: {
    status: 'upgrade',
    canUpgrade: true,
    upgradePending: false,
    onUpgrade: fn(),
    onViewBilling: fn(),
  },
};

const adminOffArgs: LabsPreviewProps = {
  status: 'ready',
  flags: off,
  pending: false,
  onToggle: fn(),
};

export const SettingsNav: Story = {
  args: adminOffArgs,
  render: (args) => (
    <SettingsFrame>
      <InteractiveLabs {...args} />
    </SettingsFrame>
  ),
};
