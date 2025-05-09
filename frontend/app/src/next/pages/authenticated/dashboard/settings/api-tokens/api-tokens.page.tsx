import { useState } from 'react';
import { ApiTokensProvider } from '@/next/hooks/use-api-tokens';
import { TokensTable } from '@/next/pages/authenticated/dashboard/settings/api-tokens/components/tokens-table';
import { Button } from '@/next/components/ui/button';
import { Separator } from '@/next/components/ui/separator';
import { Dialog } from '@/next/components/ui/dialog/dialog';
import { CreateTokenDialog } from './components/create-token-dialog';
import { DocsButton } from '@/next/components/ui/docs-button';
import useCan from '@/next/hooks/use-can';
import docs from '@/next/lib/docs';
import { apiTokens } from '@/next/lib/can/features/api-tokens.permissions';
import {
  Alert,
  AlertTitle,
  AlertDescription,
} from '@/next/components/ui/alert';
import { Lock } from 'lucide-react';
import {
  HeadlineActionItem,
  HeadlineActions,
  Headline,
  PageTitle,
} from '@/next/components/ui/page-header';
import BasicLayout from '@/next/components/layouts/basic.layout';

export default function ApiTokensPage() {
  return (
    <ApiTokensProvider>
      <ApiTokensContent />
    </ApiTokensProvider>
  );
}

function ApiTokensContent() {
  const { canWithReason } = useCan();
  const [showTokenDialog, setShowTokenDialog] = useState(false);

  const CreateTokenButton = () => (
    <Button key="create-api-token" onClick={() => setShowTokenDialog(true)}>
      Create API Token
    </Button>
  );

  const { allowed: canManage, message: canManageMessage } = canWithReason(
    apiTokens.manage(),
  );

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="API tokens are used by workers to connect with the Hatchet">
          API Tokens
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.setup} size="icon" />
          </HeadlineActionItem>
          {canManage && (
            <HeadlineActionItem>
              {canManage && <CreateTokenButton />}
            </HeadlineActionItem>
          )}
        </HeadlineActions>
      </Headline>

      {canManageMessage && (
        <Alert variant="warning">
          <Lock className="w-4 h-4 mr-2" />
          <AlertTitle>Role required</AlertTitle>
          <AlertDescription>{canManageMessage}</AlertDescription>
        </Alert>
      )}
      {canManage && (
        <>
          <Separator className="my-4" />
          <TokensTable
            emptyState={
              <div className="flex flex-col items-center justify-center gap-4 py-8">
                <p className="text-md">No API tokens found.</p>
                <p className="text-sm text-muted-foreground">
                  Create a new API token to get started.
                </p>
                {canManage && <CreateTokenButton />}
                <DocsButton doc={docs.home.setup} />
              </div>
            }
          />
        </>
      )}

      {showTokenDialog && (
        <Dialog open={showTokenDialog} onOpenChange={setShowTokenDialog}>
          <CreateTokenDialog close={() => setShowTokenDialog(false)} />
        </Dialog>
      )}
    </BasicLayout>
  );
}
