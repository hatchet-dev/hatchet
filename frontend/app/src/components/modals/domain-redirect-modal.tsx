import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';
import { getCloudMetadataQuery } from '@/hooks/use-cloud';
import type { APICloudMetadata } from '@/lib/api/generated/cloud/data-contracts';
import {
  buildRedirectFrontendHref,
  domainRedirectSkipStorageKey,
  parseRedirectFrontendOrigin,
} from '@/lib/redirect-frontend-host';
import { useQuery } from '@tanstack/react-query';
import { useCallback, useMemo, useState } from 'react';

type CloudMetadataQueryData = APICloudMetadata & { isCloudEnabled?: boolean };

function isCloudEnabledMetadata(data: unknown): data is CloudMetadataQueryData {
  return (
    typeof data === 'object' &&
    data !== null &&
    (data as CloudMetadataQueryData).isCloudEnabled === true
  );
}

export function DomainRedirectModal() {
  const { data } = useQuery(getCloudMetadataQuery);
  const [dismissed, setDismissed] = useState(false);

  const targetOrigin = useMemo(() => {
    if (!isCloudEnabledMetadata(data)) {
      return null;
    }
    const raw = data.redirectFrontendHost?.trim();
    if (!raw) {
      return null;
    }
    return parseRedirectFrontendOrigin(raw);
  }, [data]);

  const skipKey = targetOrigin?.origin
    ? domainRedirectSkipStorageKey(targetOrigin.origin)
    : null;

  const shouldOpen = useMemo(() => {
    if (dismissed || typeof window === 'undefined') {
      return false;
    }
    if (!targetOrigin || !skipKey) {
      return false;
    }
    if (window.location.origin === targetOrigin.origin) {
      return false;
    }
    try {
      if (sessionStorage.getItem(skipKey)) {
        return false;
      }
    } catch {
      return false;
    }
    return true;
  }, [dismissed, skipKey, targetOrigin]);

  const takeMeThere = useCallback(() => {
    if (!targetOrigin) {
      return;
    }
    window.location.assign(
      buildRedirectFrontendHref(targetOrigin, window.location),
    );
  }, [targetOrigin]);

  const skip = useCallback(() => {
    if (skipKey) {
      try {
        sessionStorage.setItem(skipKey, '1');
      } catch {
        // ignore quota / private mode
      }
    }
    setDismissed(true);
  }, [skipKey]);

  return (
    <Dialog open={shouldOpen} onOpenChange={(open) => !open && skip()}>
      <DialogContent className="max-w-lg text-center">
        <div className="flex flex-col items-center gap-6">
          <HatchetLogo variant="mark" className="h-10 w-10" />
          <div className="space-y-2">
            <DialogTitle className="text-center text-2xl">
              We moved Hatchet Cloud
            </DialogTitle>
            <DialogDescription className="text-center text-base text-muted-foreground">
              Your dashboard now lives on a new domain. Continue there to keep
              the same page you&apos;re on, or skip to stay here for this
              session.
            </DialogDescription>
          </div>
          <div className="flex w-full flex-col gap-2">
            <Button className="w-full" onClick={takeMeThere}>
              Take me there
            </Button>
            <Button variant="ghost" className="w-full" onClick={skip}>
              Skip
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
