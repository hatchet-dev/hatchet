import { hatchet } from '../../hatchet-client';
import { mockClassify, mockReply } from './mock-classifier';

type MessageInput = { message: string };

// > Step 01 Classify Task
const classifyTask = hatchet.durableTask({
  name: 'classify-message',
  fn: async (input: MessageInput) => {
    return { category: mockClassify(input.message) };
  },
});
// !!

// > Step 02 Specialist Tasks
const supportTask = hatchet.durableTask({
  name: 'handle-support',
  fn: async (input: MessageInput) => {
    return { response: mockReply(input.message, 'support'), category: 'support' };
  },
});

const salesTask = hatchet.durableTask({
  name: 'handle-sales',
  fn: async (input: MessageInput) => {
    return { response: mockReply(input.message, 'sales'), category: 'sales' };
  },
});

const defaultTask = hatchet.durableTask({
  name: 'handle-default',
  fn: async (input: MessageInput) => {
    return { response: mockReply(input.message, 'other'), category: 'other' };
  },
});
// !!

// > Step 03 Router Task
const routerTask = hatchet.durableTask({
  name: 'message-router',
  executionTimeout: '2m',
  fn: async (input: MessageInput) => {
    const { category } = await classifyTask.run(input);

    switch (category) {
      case 'support':
        return supportTask.run(input);
      case 'sales':
        return salesTask.run(input);
      default:
        return defaultTask.run(input);
    }
  },
});
// !!

export { classifyTask, supportTask, salesTask, defaultTask, routerTask };
