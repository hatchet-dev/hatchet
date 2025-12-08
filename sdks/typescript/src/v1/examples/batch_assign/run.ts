/* eslint-disable no-console */
import { batch } from './task';

async function main() {
  const task1 = batch.run([
    {
      Message: 'task1',
      batchId: '1',
    },
    {
      Message: 'task2',
      batchId: '1',
    },
    {
      Message: 'task3',
      batchId: '1',
    },
    {
      Message: 'task4',
      batchId: '1',
    },
    {
      Message: 'task5',
      batchId: '1',
    },
    {
      Message: 'task6',
      batchId: '1',
    },
    {
      Message: 'task7',
      batchId: '1',
    },
    {
      Message: 'task8',
      batchId: '1',
    },
    {
      Message: 'task9',
      batchId: '1',
    },
    {
      Message: 'task10',
      batchId: '2',
    },
  ]);
  const results = await task1;
  console.log(results[0].TransformedMessage);
  // console.log(results[1].TransformedMessage);
  // console.log(results[2].TransformedMessage);
  // console.log(results[3].TransformedMessage);
}

main()
  .catch(console.error)
  .finally(() => {
    console.log('done');
    process.exit(0);
  });
