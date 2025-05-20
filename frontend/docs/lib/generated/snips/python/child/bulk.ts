import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import asyncio\n\n# > Running a Task\nfrom examples.child.worker import SimpleInput, child_task\n\nchild_task.run(SimpleInput(message=\"Hello, World!\"))\n\n\nasync def main() -> None:\n    # > Bulk Run a Task\n    greetings = [\"Hello, World!\", \"Hello, Moon!\", \"Hello, Mars!\"]\n\n    results = await child_task.aio_run_many(\n        [\n            # run each greeting as a task in parallel\n            child_task.create_bulk_run_item(\n                input=SimpleInput(message=greeting),\n            )\n            for greeting in greetings\n        ]\n    )\n\n    # this will await all results and return a list of results\n    print(results)\n\n    # > Running Multiple Tasks\n    result1 = child_task.aio_run(SimpleInput(message=\"Hello, World!\"))\n    result2 = child_task.aio_run(SimpleInput(message=\"Hello, Moon!\"))\n\n    #  gather the results of the two tasks\n    gather_results = await asyncio.gather(result1, result2)\n\n    #  print the results of the two tasks\n    print(gather_results[0][\"transformed_message\"])\n    print(gather_results[1][\"transformed_message\"])\n",
  "source": "out/python/child/bulk.py",
  "blocks": {
    "running_a_task": {
      "start": 4,
      "stop": 6
    },
    "bulk_run_a_task": {
      "start": 11,
      "stop": 24
    },
    "running_multiple_tasks": {
      "start": 27,
      "stop": 35
    }
  },
  "highlights": {}
};

export default snippet;
