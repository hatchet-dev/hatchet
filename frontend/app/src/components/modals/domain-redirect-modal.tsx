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
  parseRedirectFrontendOrigin,
} from '@/lib/redirect-frontend-host';
import { useQuery } from '@tanstack/react-query';
import { useCallback, useMemo } from 'react';

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

  const targetDomain = targetOrigin?.host ?? 'a new domain';
  const targetHref =
    targetOrigin && typeof window !== 'undefined'
      ? buildRedirectFrontendHref(targetOrigin, window.location)
      : targetOrigin?.origin;

  const shouldOpen = useMemo(() => {
    if (typeof window === 'undefined') {
      return false;
    }
    if (!targetOrigin) {
      return false;
    }
    if (window.location.origin === targetOrigin.origin) {
      return false;
    }
    return true;
  }, [targetOrigin]);

  const takeMeThere = useCallback(() => {
    if (!targetOrigin) {
      return;
    }
    window.location.assign(
      buildRedirectFrontendHref(targetOrigin, window.location),
    );
  }, [targetOrigin]);

  return (
    <Dialog open={shouldOpen}>
      <DialogContent
        className="max-w-lg text-center"
        onEscapeKeyDown={(event) => event.preventDefault()}
        onInteractOutside={(event) => event.preventDefault()}
      >
        <div className="flex flex-col items-center gap-6">
          <HatchetLogo variant="mark" className="h-10 w-10" />
          <div className="space-y-2">
            <DialogTitle className="text-center text-2xl">
              We moved Hatchet Cloud
            </DialogTitle>
            <DialogDescription className="text-center text-base text-muted-foreground">
              Your dashboard now lives on{' '}
              <a
                href={targetHref}
                className="text-primary underline-offset-4 hover:underline"
              >
                {targetDomain}
              </a>
              . <br />
              <br /> We&apos;re making this change to unlock new features like
              multi-region support and improved performance.
            </DialogDescription>
          </div>
          <div className="flex w-full flex-col gap-2">
            <Button className="w-full" onClick={takeMeThere}>
              Take me there &rarr;
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
