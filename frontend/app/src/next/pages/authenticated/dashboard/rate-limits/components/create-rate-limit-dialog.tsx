import { useState } from 'react';
import { Button } from '@/next/components/ui/button';
import { Input } from '@/next/components/ui/input';
import {
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/next/components/ui/dialog';
import { useRateLimitsContext } from '@/next/hooks/use-ratelimits';

type CreateRateLimitDialogProps = {
  close: () => void;
};

export function CreateRateLimitDialog({ close }: CreateRateLimitDialogProps) {
  const { create } = useRateLimitsContext();
  const [newRateLimit, setNewRateLimit] = useState({
    key: '',
    limitValue: 100,
    window: 'MINUTE',
  });

  const handleCreateRateLimit = async () => {
    try {
      await create.mutateAsync(newRateLimit);
      close();
    } catch (error) {
      console.error('Failed to create rate limit:', error);
    }
  };

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>Create New Rate Limit</DialogTitle>
        <DialogDescription>
          Set up a new rate limit for your resources
        </DialogDescription>
      </DialogHeader>
      <div className="grid gap-4 py-4">
        <div className="grid grid-cols-4 items-center gap-4">
          <label className="text-right text-sm" htmlFor="key">
            Key
          </label>
          <Input
            id="key"
            value={newRateLimit.key}
            onChange={(e) =>
              setNewRateLimit({
                ...newRateLimit,
                key: e.target.value,
              })
            }
            className="col-span-3"
          />
        </div>
        <div className="grid grid-cols-4 items-center gap-4">
          <label className="text-right text-sm" htmlFor="limitValue">
            Limit
          </label>
          <Input
            id="limitValue"
            type="number"
            value={newRateLimit.limitValue}
            onChange={(e) =>
              setNewRateLimit({
                ...newRateLimit,
                limitValue: parseInt(e.target.value, 10),
              })
            }
            className="col-span-3"
          />
        </div>
        <div className="grid grid-cols-4 items-center gap-4">
          <label className="text-right text-sm" htmlFor="window">
            Window
          </label>
          <select
            id="window"
            value={newRateLimit.window}
            onChange={(e) =>
              setNewRateLimit({
                ...newRateLimit,
                window: e.target.value,
              })
            }
            className="col-span-3 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <option value="SECOND">Second</option>
            <option value="MINUTE">Minute</option>
            <option value="HOUR">Hour</option>
            <option value="DAY">Day</option>
          </select>
        </div>
      </div>
      <DialogFooter>
        <Button variant="outline" onClick={close}>
          Cancel
        </Button>
        <Button onClick={handleCreateRateLimit}>Create</Button>
      </DialogFooter>
    </DialogContent>
  );
}
