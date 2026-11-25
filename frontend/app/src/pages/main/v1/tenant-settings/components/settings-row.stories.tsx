import { ReadOnlyValue, SettingRow } from './settings-row';
import type { Meta, StoryObj } from '@storybook/react-vite';

const meta = {
  title: 'Pages/Tenant Settings/ReadOnlyValue',
  component: ReadOnlyValue,
  // Fullscreen so the responsive max-width breakpoints follow the viewport
  // toolbar instead of the default padded canvas.
  parameters: { layout: 'fullscreen' },
  args: {
    value:
      'https://hatchet-internal.us-west-2.onhatchet.run/api/v1/tenants/707d0855-80ab-4e1f-a156-f1c4546cbf52',
  },
  render: (args) => (
    <div className="divide-y divide-border">
      <SettingRow
        label="API URL"
        description="The base URL for this tenant's REST API."
      >
        <ReadOnlyValue {...args} />
      </SettingRow>
      <SettingRow
        label="Region"
        description="The control-plane region this tenant is deployed to."
      >
        <ReadOnlyValue value="us-west-2" />
      </SettingRow>
    </div>
  ),
} satisfies Meta<typeof ReadOnlyValue>;

export default meta;
type Story = StoryObj<typeof meta>;

export const LongValue: Story = {};

export const ShortValue: Story = {
  args: { value: 'https://cloud.onhatchet.run' },
};
