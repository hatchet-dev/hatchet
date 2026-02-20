// > Create client
import { HatchetClient } from '@hatchet/v1';

export const hatchetWithMiddleware = HatchetClient.init({
    middleware: {
        pre: [
            (_input, _ctx) => {
                return { data: 1 };
            },
            (_input, _ctx) => {
                return { requestId: 'abc-123' };
            },
        ],
        post: (_output, _ctx, _input) => {
            return { extra: 2 };
        },
    },
});

// !!


// > Chaining middleware
export const hatchetWithMiddlewareChaining = HatchetClient.init({
    middleware: {
        pre: [
            (_input, _ctx) => {
                return { first: 1 };
            },
            (_input, _ctx) => {
                return { second: 2 };
            },
        ],
    },
});

// !!
