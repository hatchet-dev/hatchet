import { Separator } from '@/components/ui/separator';
import { TenantContextType } from '@/lib/outlet';
import { useOutletContext } from 'react-router-dom';
import { Label } from '@/components/ui/label.tsx';
import { Input } from '@/components/ui/input.tsx';
import CopyToClipboard from '@/components/ui/copy-to-clipboard.tsx';

export default function Webhooks() {
  const { tenant } = useOutletContext<TenantContextType>();

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <h2 className="text-2xl font-semibold leading-tight text-foreground">
            Webhooks
          </h2>
        </div>
        <p className="text-gray-700 dark:text-gray-300 my-4">
          Settings and signing keys related to webhook workflows.
        </p>
        <Separator className="my-4" />

        <div className="grid gap-2 grid-cols-1 sm:grid-cols-2">
          <div className="">
            <Label htmlFor="email">Webhook secret</Label>
            <Input
              id="secret"
              placeholder="Secret"
              autoCapitalize="none"
              autoCorrect="off"
              className=" min-w-[300px]"
              disabled
              value={tenant.webhookSecret || 'n/a'}
            />
            {tenant.webhookSecret && (
              <CopyToClipboard text={tenant.webhookSecret} withText />
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
