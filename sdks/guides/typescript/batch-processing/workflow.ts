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
const parentTask = hatchet.durableTask<BatchInput>({
  name: 'spawn-children',
  fn: async (input) => {
    const results = await Promise.all(
      input.items.map((itemId) => childTask.run({ item_id: itemId }))
    );
    return { processed: results.length, results };
  },
});
// !!

// > Step 03 Process Item
function processItem(input: ItemInput) {
  return { status: 'done', item_id: input.item_id };
}
// !!

export { parentTask, childTask };
