import {
  workflowLanguageOptions,
  type WorkflowLanguageKey,
} from './onboarding-options';
import { Tabs, TabsList, TabsTrigger } from '@/components/v1/ui/tabs';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';

// The SDK is a global preference (survives across tenants), so it lives
// under its own key rather than the tenant-scoped onboarding state.
const preferredSdkStorageKey = 'hatchet:preferred-sdk';

export type Sdk = WorkflowLanguageKey;

const defaultSdk: Sdk = 'python';

// A malformed or unknown stored value falls back to Python rather than
// propagating garbage.
function normalizeSdk(value: unknown): Sdk {
  return typeof value === 'string' && value in workflowLanguageOptions
    ? (value as Sdk)
    : defaultSdk;
}

export function usePreferredSdk(): [Sdk, (next: Sdk) => void] {
  const [stored, setStored] = useLocalStorageState<Sdk>(
    preferredSdkStorageKey,
    defaultSdk,
  );

  return [normalizeSdk(stored), (next: Sdk) => setStored(normalizeSdk(next))];
}

// The shared Button and Tabs remove the native focus outline without a
// replacement, so focusable onboarding controls carry an explicit ring.
export const focusRing =
  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ring-offset-background';

export const segmentedTabsListClass =
  'bg-muted ring-1 ring-border/50 rounded-lg p-0 gap-0.5 dark:bg-muted/20 dark:ring-inset';
export const segmentedTabsTriggerClass = `rounded-lg h-full text-muted-foreground data-[state=active]:ring-1 data-[state=active]:ring-border data-[state=active]:bg-background dark:data-[state=active]:bg-muted/70 dark:data-[state=active]:shadow-lg dark:ring-inset ${focusRing}`;

export function SdkSwitcher({
  value,
  onChange,
}: {
  value: Sdk;
  onChange: (next: Sdk) => void;
}) {
  return (
    <Tabs
      value={value}
      onValueChange={(next) => onChange(next as Sdk)}
      className="w-fit"
    >
      <TabsList className={segmentedTabsListClass}>
        {Object.values(workflowLanguageOptions).map((option) => (
          <TabsTrigger
            key={option.value}
            value={option.value}
            className={segmentedTabsTriggerClass}
          >
            {option.label}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  );
}
