// > Init a client with middleware
import { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';


// These types will be merged into all task input and output types
export type GlobalInputType = {
    first: number;
    second: number;
};

export type GlobalOutputType = {
    extra: number;
};

export const hatchetWithMiddleware = HatchetClient.init<GlobalInputType, GlobalOutputType>()
    .withMiddleware({
        // This middleware will be run before every task
        pre: (input, ctx) => {
            input.first;
            return { ...input, dependency: 'abc-123' };
        },
        // This middleware will be run after every task
        post: (output, ctx, input) => {
            return { ...output, additionalData: 2 };
        },
    });



// > Chaining middleware
export const hatchetWithMiddlewareChaining = HatchetClient.init<GlobalInputType>()
    .withMiddleware({
        pre: (input, ctx) => {
            input.first;
            return { ...input, dependency: 'abc-123' };
        },
        post: (output, ctx, input) => {
            return { ...output, firstExtra: 3 };
        },
    })
    .withMiddleware({
        pre: (input, ctx) => {
            input.dependency; // available from previous middleware
            return { ...input, anotherDep: true };
        },
        post: (output, ctx, input) => {
            return { ...output, secondExtra: 4 };
        },
    });
