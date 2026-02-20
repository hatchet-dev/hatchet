// > Create client
import { HatchetClient } from '@hatchet/v1';


export type GlobalInputType = {
    first: number;
    second: number;
};

export type GlobalOutputType = {
    extra: number;
};

export const hatchetWithoutMiddleware = HatchetClient.init<GlobalInputType, GlobalOutputType>()



export const hatchetWithMiddleware = hatchetWithoutMiddleware.withMiddleware({
        pre: (_input, _ctx) => {
            _input.first;
            return { requestId: 'abc-123' };
        },
        post: (_output, _ctx, _input) => {
            return { extra: 2 };
        },
    }).withMiddleware({
        pre: (_input, _ctx) => {
            _input.first;
            return { xxx: 'abc-123' };
        },
        post: (_output, _ctx, _input) => {
            return { extra: 2 };
        },
    });
// !!



// > Chaining middleware
export const hatchetWithMiddlewareChaining = HatchetClient.init<GlobalInputType>()
    .withMiddleware({
        pre: [
            (_input, _ctx) => {
                _input.first;
                return { first: 1 };
            },
            (_input, _ctx) => {
                return { second: 2 };
            },
        ],
    });

// !!
