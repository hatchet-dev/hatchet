import type { UnaryRequest } from '@connectrpc/connect';
import { createAuthInterceptor } from './transport';

async function headerSetBy(token: string): Promise<Headers> {
  const request = { header: new Headers() } as UnaryRequest;
  await createAuthInterceptor(token)(async (req) => req as never)(request);
  return request.header;
}

describe('createAuthInterceptor', () => {
  it('sends the token as a bearer authorization header', async () => {
    expect((await headerSetBy('eyJhbGciOi.eyJzdWIi.sig')).get('authorization')).toBe(
      'Bearer eyJhbGciOi.eyJzdWIi.sig'
    );
  });

  it('refuses a header-unsafe token with a message that does not contain it', () => {
    const token = 'eyJhbGciOi.eyJzdWIi.sig\ncontinuation';

    let error: Error | undefined;
    try {
      createAuthInterceptor(token);
    } catch (e) {
      error = e as Error;
    }

    expect(error).toBeDefined();
    expect(error?.message).toMatch(/cannot be sent in an HTTP header/);
    expect(error?.message).not.toContain('eyJhbGciOi');
    expect(error?.message).not.toContain('continuation');
  });
});
