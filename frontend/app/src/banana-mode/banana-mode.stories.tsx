import { BananaMode } from './banana-mode';
import { BananaModeSwitch } from './banana-mode-switch';
import { AppContext, type AppContextValue } from '@/providers/app-context';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';

const storyTenant = {
  tenantId: 'story-tenant',
} as AppContextValue;

function Stage() {
  const [clicks, setClicks] = useState(0);

  return (
    <AppContext.Provider value={storyTenant}>
      <BananaMode />
      <div className="min-h-[520px] bg-background p-8 text-foreground">
        <BananaModeSwitch />
        <button
          type="button"
          className="mt-8 rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground"
          onClick={() => setClicks((count) => count + 1)}
        >
          Clicks: {clicks}
        </button>
      </div>
    </AppContext.Provider>
  );
}

const meta = {
  title: 'Labs/BananaMode',
  component: BananaMode,
} satisfies Meta<typeof BananaMode>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Preview: Story = {
  render: () => <Stage />,
};
