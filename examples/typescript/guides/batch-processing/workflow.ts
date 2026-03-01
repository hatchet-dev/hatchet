import { hatchet } from '../../hatchet-client';

type BatchInput = { items: string[] };
type ItemInput = { item_id: string };

const childTask = hatchet.task<ItemInput>({
  name: 'process-item',
  fn: async (input) => ({
    status: 'done',
    item_id: input.item_id,
  }),
});

// > Step 01 Define Parent Task
const parentTask = hatchet.task<BatchInput>({
  name: 'spawn-children',
  fn: async (input) => {
    const results = [];
    for (const itemId of input.items) {
      const result = await childTask.run({ item_id: itemId });
      results.push(result);
    }
    return { processed: results.length, results };
  },
});

// > Step 02 Fan Out Children
async function fanOut(input: BatchInput) {
  const results: unknown[] = [];
  for (const itemId of input.items) {
    const result = await childTask.run({ item_id: itemId });
    results.push(result);
  }
  return results;
}
// Hatchet distributes child runs across available workers.

// > Step 03 Process Item
function processItem(input: ItemInput) {
  return { status: 'done', item_id: input.item_id };
}

export { parentTask, childTask };
