import { useMemo, useState } from 'react';
import { Input } from '@/next/components/ui/input';
import { Button } from '@/next/components/ui/button';
import { Label } from '@/next/components/ui/label';
import { CreateAPITokenResponse } from '@/lib/api';
import {
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/next/components/ui/dialog/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/next/components/ui/select';
import useApiTokens from '@/next/hooks/use-api-tokens';
import useUser from '@/next/hooks/use-user';
import { Code } from '@/next/components/ui/code';
import { useTenant } from '@/next/hooks/use-tenant';
interface CreateTokenDialogProps {
  onSuccess?: (data?: CreateAPITokenResponse) => void;
  close: () => void;
}

const options = {
  '100 years': '87600h',
  '1 year': '8760h',
  '30 days': '720h',
  '7 days': '168h',
  '1 day': '24h',
  '1 hour': '1h',
};

export function CreateTokenDialog({
  onSuccess,
  close,
}: CreateTokenDialogProps) {
  const { data: user } = useUser();
  const { tenant } = useTenant();

  const defaultToken = useMemo(() => {
    if (!user?.name) {
      return '';
    }

    // Format date as "apr-5-2025" (lowercase month abbreviation, day, year)
    const now = new Date();
    const month = now.toLocaleString('en-US', { month: 'short' }).toLowerCase();
    const day = now.getDate();
    const year = now.getFullYear();

    // Format as "username apr-5-2025" with no spaces or special chars
    return `${user.name.toLowerCase()}--${month}-${day}-${year}`
      .replace(/[^a-z0-9-]/g, '')
      .replace(/ /g, '-');
  }, [user?.name]);

  const [name, setName] = useState(defaultToken);
  const [expiresIn, setExpiresIn] = useState(options['100 years']); // Default 100 years

  const [token, setToken] = useState<string | null>(null);

  const { create } = useApiTokens();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const token = await create.mutateAsync({
      name,
      expiresIn,
    });
    setToken(`export HATCHET_CLIENT_TOKEN="${token.token}"`);
    onSuccess?.(token);
  };

  return (
    <DialogContent className="max-w-[600px]">
      <DialogHeader>
        <DialogTitle>
          {token ? (
            'Token Created'
          ) : (
            <>
              Create API Token for{' '}
              <span className="font-mono">{tenant?.name}</span>
            </>
          )}
        </DialogTitle>
      </DialogHeader>
      <DialogDescription>
        {token
          ? 'Your new API token has been created.'
          : 'Tokens are private and should not be shared. They are used to connect your workers to your Hatchet Tenant via SDKs and API.'}
      </DialogDescription>

      {token ? (
        <>
          <div className="mt-4 space-y-4">
            <div>
              <div className="flex mt-1 max-w-[550px] overflow-hidden">
                <Code
                  className="text-sm"
                  title="API Token"
                  language="bash"
                  value={token}
                />
              </div>
              <p className="text-sm text-muted-foreground mt-2">
                This token will only be shown once. Please copy it now.
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" type="button" onClick={() => close()}>
              Close
            </Button>
            <Button
              variant="default"
              type="button"
              onClick={() => {
                navigator.clipboard.writeText(token);
                close();
              }}
            >
              Copy and Close
            </Button>
          </DialogFooter>
        </>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <Label htmlFor="name">Token Name</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Token name"
              required
              className="mt-1"
            />
          </div>

          <div>
            <Label htmlFor="expiresIn">Expires In</Label>
            <Select value={expiresIn} onValueChange={setExpiresIn}>
              <SelectTrigger className="w-full mt-1">
                <SelectValue placeholder="Select expiration time" />
              </SelectTrigger>
              <SelectContent>
                {Object.entries(options).map(([key, value]) => (
                  <SelectItem key={value} value={value}>
                    {key}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <DialogFooter>
            <Button variant="outline" type="button" onClick={() => close()}>
              Cancel
            </Button>
            <Button variant="default" type="submit" loading={create.isPending}>
              Create Token
            </Button>
          </DialogFooter>
        </form>
      )}
    </DialogContent>
  );
}
