// > Init a client with middleware
import { HatchetClient, HatchetMiddleware } from '@hatchet-dev/typescript-sdk/v1';

export type GlobalInputType = {
  first: number;
  second: number;
};

export type GlobalOutputType = {
  extra: number;
};

const myMiddleware = {
  before: (input, ctx) => {
    console.log('before', input.first);
    return { ...input, dependency: 'abc-123' };
  },
  after: (output, ctx, input) => {
    return { ...output, additionalData: 2 };
  },
} satisfies HatchetMiddleware<GlobalInputType, GlobalOutputType>;

export const hatchetWithMiddleware = HatchetClient.init<
  GlobalInputType,
  GlobalOutputType
>().withMiddleware(myMiddleware);

// > Chaining middleware
const firstMiddleware = {
  before: (input, ctx) => {
    console.log('before', input.first);
    return { ...input, dependency: 'abc-123' };
  },
  after: (output, ctx, input) => {
    return { ...output, firstExtra: 3 };
  },
} satisfies HatchetMiddleware<GlobalInputType>;

const secondMiddleware = {
  before: (input, ctx) => {
    console.log('before', input.dependency); // available from previous middleware
    return { ...input, anotherDep: true };
  },
  after: (output, ctx, input) => {
    return { ...output, secondExtra: 4 };
  },
} satisfies HatchetMiddleware<GlobalInputType & { dependency: string }>;

export const hatchetWithMiddlewareChaining = HatchetClient.init<GlobalInputType>()
  .withMiddleware(firstMiddleware)
  .withMiddleware(secondMiddleware);
