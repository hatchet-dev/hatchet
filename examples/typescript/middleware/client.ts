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
            return { dependency: 'abc-123' };
        },
        // This middleware will be run after every task
        post: (output, ctx, input) => {
            return { additionalData: 2 };
        },
    });



// > Chaining middleware
export const hatchetWithMiddlewareChaining = HatchetClient.init<GlobalInputType>()
    .withMiddleware({
        // These middleware will be run in order before every task
        pre: [
            (input, ctx) => {
                input.first;
                return { first: 1 };
            },
            (input, ctx) => {
                return { second: 2 };
            },
        ],
        // These middleware will be run in order after every task
        post: [
            (output, ctx, input) => {
                return { firstExtra: 3 };
            },
            (output, ctx, input) => {
                return { secondExtra: 4 };
            }
        ],
    });
