// > Create client
import { HatchetClient } from '@hatchet/v1';

export const hatchetWithMiddleware = HatchetClient.init({
    middleware: {
        pre: (_input, _ctx) => {
            return { data: 1 };
        },
        post: (_output, _ctx, _input) => {
            return { extra: 2 };
        },
    },
});

type TaskInput = {
    message: string;
};

type TaskOutput = {
    message: string;
};

// Note: for type safety with middleware, we need to explicitly specify the input and output types in the generic parameters
const taskWithMiddleware = hatchetWithMiddleware.task<TaskInput, TaskOutput>({
    name: 'task-with-middleware',
    fn: (input, _ctx) => {
        console.log('task', input.data);    // number  (from pre middleware)
        console.log('task', input.message); // string  (from TaskWithMiddlewareInput)
        return { message: input.message };
    },
});


async function main() {
    const result = await taskWithMiddleware.run({
        message: 'hello',
    });

    console.log('result', result.extra);   // number  (from post middleware)
    console.log('result', result.message); // string  (from TaskWithMiddlewareOutput)
}

if (require.main === module) {
    main();
}
// !!
