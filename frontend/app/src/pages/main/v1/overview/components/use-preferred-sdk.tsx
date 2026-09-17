import { workflowLanguageOptions } from './onboarding-options';
import { Tabs, TabsList, TabsTrigger } from '@/components/v1/ui/tabs';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { cn } from '@/lib/utils';

// The SDK is a global preference (survives across tenants), so it lives
// under its own key rather than the tenant-scoped onboarding state.
export const preferredSdkStorageKey = 'hatchet:preferred-sdk';

// The broader SDK set (matches WorkerRuntimeSDKs). Ruby is early access, so
// it is a valid stored value even though the inline switcher below only
// offers the three fully supported languages.
export type Sdk = 'python' | 'typescript' | 'go' | 'ruby';

const validSdks: readonly Sdk[] = ['python', 'typescript', 'go', 'ruby'];
const defaultSdk: Sdk = 'python';

function normalizeSdk(value: unknown): Sdk {
  return typeof value === 'string' &&
    (validSdks as readonly string[]).includes(value)
    ? (value as Sdk)
    : defaultSdk;
}

// Read order: stored value, else Python. A malformed or unknown stored
// value falls back to Python rather than propagating garbage.
export function usePreferredSdk(): [Sdk, (next: Sdk) => void] {
  const [stored, setStored] = useLocalStorageState<Sdk>(
    preferredSdkStorageKey,
    defaultSdk,
  );

  const sdk = normalizeSdk(stored);
  const setSdk = (next: Sdk) => setStored(normalizeSdk(next));

  return [sdk, setSdk];
}

// The shared Button removes the native focus outline without a replacement,
// so the switcher carries an explicit ring, matching learn-workflow-section.
const focusRing =
  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ring-offset-background';

// A small SDK switcher styled like the language Tabs in
// learn-workflow-section. It offers the three fully supported languages;
// selecting one writes through to usePreferredSdk.
export function SdkSwitcher({
  value,
  onChange,
  className,
}: {
  value: Sdk;
  onChange: (next: Sdk) => void;
  className?: string;
}) {
  return (
    <Tabs
      value={value}
      onValueChange={(next) => onChange(next as Sdk)}
      className={cn('w-fit', className)}
    >
      <TabsList className="bg-muted ring-1 ring-border/50 rounded-lg p-0 gap-0.5 dark:bg-muted/20 dark:ring-inset">
        {Object.values(workflowLanguageOptions).map((option) => (
          <TabsTrigger
            key={option.value}
            value={option.value}
            className={`rounded-lg h-full text-muted-foreground data-[state=active]:text-brand data-[state=active]:font-medium data-[state=active]:ring-1 data-[state=active]:ring-border data-[state=active]:bg-background dark:data-[state=active]:bg-muted/70 dark:data-[state=active]:shadow-lg dark:ring-inset ${focusRing}`}
          >
            {option.label}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  );
}
