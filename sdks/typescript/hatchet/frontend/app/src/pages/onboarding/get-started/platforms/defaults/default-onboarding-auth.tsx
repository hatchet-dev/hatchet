import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { CodeHighlighter } from '@/components/ui/code-highlighter';
import api, { CreateAPITokenRequest, queries } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useState } from 'react';

export const DefaultOnboardingAuth: React.FC<{
  tenantId: string;
  onAuthComplete: () => void;
  skip: () => void;
}> = ({ tenantId, onAuthComplete, skip }) => {
  const [generatedToken, setGeneratedToken] = useState<string | undefined>();

  const { handleApiError } = useApiError({});

  const tokenQuery = useQuery({
    ...queries.tokens.list(tenantId),
    refetchInterval: 5000,
  });

  const createTokenMutation = useMutation({
    mutationKey: ['api-token:create', tenantId],
    mutationFn: async (data: CreateAPITokenRequest) => {
      const res = await api.apiTokenCreate(tenantId, data);
      return res.data;
    },
    onSuccess: (data) => {
      setGeneratedToken(data.token);
    },
    onError: handleApiError,
  });

  if (generatedToken) {
    return (
      <div>
        <p className="mb-4">
          In the root of your project, create a new file called{' '}
          <Badge variant="secondary">.env</Badge>. Paste the secret token into
          this file.
        </p>
        <p className="mb-4">
          This is the only time we will show you this auth token. Make sure to
          copy it somewhere safe and do not share it with others. You can manage
          your auth tokens from the settings page.
        </p>
        <div className="rounded-lg p-4 mb-6" onClick={onAuthComplete}>
          <CodeHighlighter
            language="plaintext"
            className="text-sm"
            wrapLines={false}
            code={'HATCHET_CLIENT_TOKEN="' + generatedToken + '"'}
            copy
          />
        </div>

        <p className="text-gray-400">
          Make sure to save the <Badge variant="secondary">.env</Badge> file
          after pasting the token.
        </p>
      </div>
    );
  }

  return (
    <div>
      <p className="mb-4">
        Before you can start your worker, you need to generate an auth token.
      </p>
      <Button
        onClick={() => createTokenMutation.mutate({ name: 'default' })}
        className="mr-2"
        variant="default"
        size={'lg'}
      >
        Generate Auth Token
      </Button>
      {(tokenQuery.data?.rows ?? []).length > 0 && (
        <Button onClick={() => skip()} size={'lg'} variant="outline">
          Skip
        </Button>
      )}
    </div>
  );
};
